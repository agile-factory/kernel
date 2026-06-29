export type ConfigField = {
  name: string
  type: string
  required: boolean
  description: string
}

export type MethodDoc = {
  name: string
  signature: string
  description: string
  example?: string
}

export type EventDoc = {
  name: string
  direction: 'Published' | 'Consumed'
  description: string
  payload: string
}

export type EndpointDoc = {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  path: string
  auth: string
  body?: string
  responses: { status: string; body: string }[]
}

export type MigrationDoc = { file: string; description: string }

export type ModuleDoc = {
  slug: string
  name: string
  title: string
  description: string
  overview: string
  install: string
  config: ConfigField[]
  usage: string
  methods: MethodDoc[]
  events: EventDoc[]
  endpoints: EndpointDoc[]
  migrations: MigrationDoc[]
}

const audit: ModuleDoc = {
  slug: 'audit',
  name: 'audit',
  title: 'audit',
  description:
    'Append-only event logging with v1 offset and v2 cursor pagination, plus at-least-once outbox delivery to downstream consumers.',
  overview:
    'The audit module records an immutable, append-only trail of everything that happens inside a tenant. Events are written in the same transaction as the change that produced them, then delivered at-least-once to registered consumers through a transactional outbox. Reads support both legacy v1 offset pagination and the recommended v2 cursor pagination, which stays stable under concurrent writes.',
  install: `import "github.com/agile-factory/kernel/audit"

func migrate(db *sql.DB) error {
	return migrator.Run(db, audit.Migrations)
}`,
  config: [
    {
      name: 'DB',
      type: '*sql.DB',
      required: true,
      description: 'The database handle used for writes, reads, and the outbox.',
    },
    {
      name: 'OutboxBatchSize',
      type: 'int',
      required: false,
      description: 'Rows drained per outbox tick. Defaults to 100.',
    },
    {
      name: 'OutboxInterval',
      type: 'time.Duration',
      required: false,
      description: 'How often the outbox worker polls. Defaults to 2s.',
    },
    {
      name: 'Now',
      type: 'func() time.Time',
      required: false,
      description: 'Clock injection for deterministic tests. Defaults to time.Now.',
    },
  ],
  usage: `auditMod := audit.New(audit.Config{
	DB:              db,
	OutboxBatchSize: 200,
	OutboxInterval:  time.Second,
})

// Register a consumer that receives every event at-least-once.
auditMod.Subscribe(func(ctx context.Context, e audit.Event) error {
	return analytics.Track(ctx, e)
})

// Start the background outbox worker.
go auditMod.RunOutbox(ctx)`,
  methods: [
    {
      name: 'Log',
      signature:
        'func (m *Module) Log(ctx context.Context, e EventInput) (Event, error)',
      description:
        'Appends a single event. Call within your business transaction by passing a *sql.Tx via context so the event and the change commit atomically.',
      example: `evt, err := auditMod.Log(ctx, audit.EventInput{
	TenantID: tenantID,
	ActorID:  userID,
	Action:   "tenant.settings.updated",
	Target:   "tnt_01HF...",
	Metadata: map[string]any{"field": "billing_email"},
})`,
    },
    {
      name: 'ListV2',
      signature:
        'func (m *Module) ListV2(ctx context.Context, q CursorQuery) (Page[Event], error)',
      description:
        'Returns a page of events using cursor pagination. The returned NextCursor is opaque and stable across inserts. This is the recommended read path.',
      example: `page, err := auditMod.ListV2(ctx, audit.CursorQuery{
	TenantID: tenantID,
	Limit:    50,
	Cursor:   req.URL.Query().Get("cursor"),
})
// page.Items, page.NextCursor`,
    },
    {
      name: 'List',
      signature:
        'func (m *Module) List(ctx context.Context, q OffsetQuery) (Page[Event], error)',
      description:
        'Legacy v1 offset pagination. Retained for backwards compatibility; prefer ListV2 for new code as offsets drift under concurrent writes.',
    },
    {
      name: 'Subscribe',
      signature: 'func (m *Module) Subscribe(h Handler)',
      description:
        'Registers a consumer invoked for every event via the outbox. Handlers must be idempotent because delivery is at-least-once.',
    },
    {
      name: 'RunOutbox',
      signature: 'func (m *Module) RunOutbox(ctx context.Context) error',
      description:
        'Starts the background worker that drains the outbox and fans events out to subscribers. Blocks until the context is cancelled.',
    },
  ],
  events: [
    {
      name: 'audit.event.recorded',
      direction: 'Published',
      description:
        'Emitted through the outbox for every appended event. Consumers receive this at-least-once.',
      payload: `{
  "id": "evt_01HF...",
  "tenant_id": "tnt_01HF...",
  "actor_id": "usr_01HF...",
  "action": "tenant.settings.updated",
  "target": "tnt_01HF...",
  "metadata": { "field": "billing_email" },
  "recorded_at": "2026-06-29T10:12:04Z"
}`,
    },
  ],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/audit/events',
      auth: 'Bearer (member)',
      responses: [
        {
          status: '200',
          body: `{ "items": [ { "id", "action", "actor_id", "recorded_at" } ], "next_cursor": "c_8f2c..." }`,
        },
      ],
    },
    {
      method: 'GET',
      path: '/api/v1/audit/events/{id}',
      auth: 'Bearer (member)',
      responses: [
        { status: '200', body: `{ "id", "action", "target", "metadata", "recorded_at" }` },
        { status: '404', body: `{ "error": "event_not_found" }` },
      ],
    },
  ],
  migrations: [
    {
      file: '0001_create_audit_events.sql',
      description: 'Append-only audit_events table with tenant + recorded_at index.',
    },
    {
      file: '0002_create_audit_outbox.sql',
      description: 'Transactional outbox table and delivery cursor.',
    },
    {
      file: '0003_add_cursor_index.sql',
      description: 'Composite (tenant_id, id) index backing v2 cursor pagination.',
    },
  ],
}

const auth: ModuleDoc = {
  slug: 'auth',
  name: 'auth',
  title: 'auth',
  description:
    'Owner signup, login, refresh, logout, email verification, and invite acceptance — JWT access tokens with rotating refresh tokens.',
  overview:
    'The auth module handles the full owner authentication lifecycle. Signup provisions the first owner and their tenant atomically. Sessions use short-lived JWT access tokens paired with rotating, revocable refresh tokens. Every sensitive action emits an audit event when an audit module is wired in.',
  install: `import "github.com/agile-factory/kernel/auth"

func migrate(db *sql.DB) error {
	return migrator.Run(db, auth.Migrations)
}`,
  config: [
    { name: 'DB', type: '*sql.DB', required: true, description: 'Database handle.' },
    {
      name: 'JWTSecret',
      type: 'string',
      required: true,
      description: 'HMAC secret used to sign access tokens.',
    },
    {
      name: 'Audit',
      type: '*audit.Module',
      required: false,
      description: 'When set, auth actions are recorded to the audit trail.',
    },
    {
      name: 'AccessTTL',
      type: 'time.Duration',
      required: false,
      description: 'Access token lifetime. Defaults to 15m.',
    },
  ],
  usage: `authMod := auth.New(auth.Config{
	DB:        db,
	JWTSecret: os.Getenv("JWT_SECRET"),
	Audit:     auditMod,
})
authMod.Mount(mux, "/api/v1/auth")`,
  methods: [
    {
      name: 'SignupOwner',
      signature:
        'func (m *Module) SignupOwner(ctx context.Context, in SignupInput) (Session, error)',
      description:
        'Creates the first owner and their tenant in one transaction, returning a fresh session.',
    },
    {
      name: 'Login',
      signature: 'func (m *Module) Login(ctx context.Context, in Credentials) (Session, error)',
      description: 'Verifies credentials and issues a new access + refresh token pair.',
    },
    {
      name: 'Refresh',
      signature: 'func (m *Module) Refresh(ctx context.Context, token string) (Session, error)',
      description: 'Rotates a refresh token, revoking the old one and issuing a new pair.',
    },
    {
      name: 'Logout',
      signature: 'func (m *Module) Logout(ctx context.Context, token string) error',
      description: 'Revokes the supplied refresh token.',
    },
    {
      name: 'VerifyEmail',
      signature: 'func (m *Module) VerifyEmail(ctx context.Context, token string) error',
      description: 'Marks an email address as verified from a one-time token.',
    },
    {
      name: 'AcceptInvite',
      signature:
        'func (m *Module) AcceptInvite(ctx context.Context, in AcceptInviteInput) (Session, error)',
      description: 'Consumes a membership invite and creates the invited user account.',
    },
  ],
  events: [
    {
      name: 'auth.owner.signed_up',
      direction: 'Published',
      description: 'Emitted when a new owner and tenant are provisioned.',
      payload: `{ "user_id": "usr_01HF...", "tenant_id": "tnt_01HF...", "email": "owner@acme.io" }`,
    },
    {
      name: 'auth.user.logged_in',
      direction: 'Published',
      description: 'Emitted on every successful login.',
      payload: `{ "user_id": "usr_01HF...", "ip": "203.0.113.4", "at": "2026-06-29T10:00:00Z" }`,
    },
  ],
  endpoints: [
    {
      method: 'POST',
      path: '/api/v1/auth/signup',
      auth: 'none',
      body: `{ "email", "password", "tenant_name" }`,
      responses: [
        { status: '201', body: `{ "access_token", "refresh_token", "user" }` },
        { status: '409', body: `{ "error": "email_taken" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/login',
      auth: 'none',
      body: `{ "email", "password" }`,
      responses: [
        { status: '200', body: `{ "access_token", "refresh_token", "user" }` },
        { status: '401', body: `{ "error": "invalid_credentials" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/refresh',
      auth: 'none',
      body: `{ "refresh_token" }`,
      responses: [{ status: '200', body: `{ "access_token", "refresh_token" }` }],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/logout',
      auth: 'Bearer',
      body: `{ "refresh_token" }`,
      responses: [{ status: '204', body: `(no content)` }],
    },
  ],
  migrations: [
    { file: '0001_create_users.sql', description: 'Users table with hashed credentials.' },
    {
      file: '0002_create_refresh_tokens.sql',
      description: 'Rotating refresh-token store with revocation.',
    },
    {
      file: '0003_create_email_verifications.sql',
      description: 'One-time email verification + invite tokens.',
    },
  ],
}

/** Generic builder for the remaining modules so every slug resolves. */
function genericModule(
  partial: Pick<ModuleDoc, 'slug' | 'name' | 'title' | 'description' | 'overview'> &
    Partial<ModuleDoc>,
): ModuleDoc {
  const base = partial.name
  return {
    install: `import "github.com/agile-factory/kernel/${partial.slug}"

func migrate(db *sql.DB) error {
	return migrator.Run(db, ${base}.Migrations)
}`,
    config: [
      { name: 'DB', type: '*sql.DB', required: true, description: 'Database handle.' },
      {
        name: 'Audit',
        type: '*audit.Module',
        required: false,
        description: 'Optional audit trail integration.',
      },
    ],
    usage: `${base}Mod := ${base}.New(${base}.Config{DB: db, Audit: auditMod})
${base}Mod.Mount(mux, "/api/v1/${partial.slug}")`,
    methods: [],
    events: [],
    endpoints: [],
    migrations: [
      {
        file: `0001_create_${base}.sql`,
        description: `Core ${base} tables scoped by tenant_id.`,
      },
    ],
    ...partial,
  }
}

const others: ModuleDoc[] = [
  genericModule({
    slug: 'tenant',
    name: 'tenant',
    title: 'tenant',
    description: 'Tenant CRUD and per-tenant settings.',
    overview:
      'The tenant module manages the lifecycle of tenants and their settings. Tenants are the top-level isolation boundary; every other module scopes its data by tenant_id.',
    methods: [
      {
        name: 'Create',
        signature: 'func (m *Module) Create(ctx context.Context, in CreateInput) (Tenant, error)',
        description: 'Creates a tenant and seeds default settings.',
      },
      {
        name: 'Get',
        signature: 'func (m *Module) Get(ctx context.Context, id string) (Tenant, error)',
        description: 'Fetches a tenant by id.',
      },
      {
        name: 'UpdateSettings',
        signature:
          'func (m *Module) UpdateSettings(ctx context.Context, id string, s TenantSettings) error',
        description: 'Patches per-tenant settings.',
      },
    ],
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/tenants',
        auth: 'Bearer (owner)',
        body: `{ "name" }`,
        responses: [{ status: '201', body: `{ "id", "name", "settings" }` }],
      },
      {
        method: 'PATCH',
        path: '/api/v1/tenants/{id}/settings',
        auth: 'Bearer (owner)',
        body: `{ "settings": { ... } }`,
        responses: [{ status: '200', body: `{ "id", "settings" }` }],
      },
    ],
  }),
  genericModule({
    slug: 'membership',
    name: 'membership',
    title: 'membership',
    description: 'Invite, revoke, and list members of a tenant.',
    overview:
      'The membership module governs who belongs to a tenant and in what role. Invites are tokenized and consumed by the auth module on acceptance.',
    methods: [
      {
        name: 'Invite',
        signature: 'func (m *Module) Invite(ctx context.Context, in InviteInput) (Invite, error)',
        description: 'Creates an invite and emits a notification event.',
      },
      {
        name: 'Revoke',
        signature: 'func (m *Module) Revoke(ctx context.Context, memberID string) error',
        description: 'Removes a member from the tenant.',
      },
      {
        name: 'List',
        signature: 'func (m *Module) List(ctx context.Context, q CursorQuery) (Page[Member], error)',
        description: 'Lists members with cursor pagination.',
      },
    ],
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/members/invites',
        auth: 'Bearer (admin)',
        body: `{ "email", "role" }`,
        responses: [{ status: '201', body: `{ "invite_id", "email", "role" }` }],
      },
      {
        method: 'DELETE',
        path: '/api/v1/members/{id}',
        auth: 'Bearer (admin)',
        responses: [{ status: '204', body: `(no content)` }],
      },
    ],
  }),
  genericModule({
    slug: 'billing',
    name: 'billing',
    title: 'billing',
    description: 'Stripe webhook, Wave stub, plans, and subscriptions.',
    overview:
      'The billing module manages plans and subscriptions and reconciles state from payment providers. Stripe is supported via signed webhooks; a Wave stub is provided for African payment rails.',
    methods: [
      {
        name: 'HandleStripeWebhook',
        signature: 'func (m *Module) HandleStripeWebhook(w http.ResponseWriter, r *http.Request)',
        description: 'Verifies the Stripe signature and reconciles subscription state.',
      },
      {
        name: 'Subscribe',
        signature:
          'func (m *Module) Subscribe(ctx context.Context, in SubscribeInput) (Subscription, error)',
        description: 'Starts a subscription for a tenant on a given plan.',
      },
    ],
    events: [
      {
        name: 'billing.subscription.activated',
        direction: 'Published',
        description: 'Emitted when a subscription becomes active.',
        payload: `{ "tenant_id", "plan", "status": "active" }`,
      },
    ],
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/billing/webhooks/stripe',
        auth: 'Stripe signature',
        responses: [{ status: '200', body: `{ "received": true }` }],
      },
      {
        method: 'POST',
        path: '/api/v1/billing/subscriptions',
        auth: 'Bearer (owner)',
        body: `{ "plan" }`,
        responses: [{ status: '201', body: `{ "subscription_id", "status" }` }],
      },
    ],
  }),
  genericModule({
    slug: 'notifications',
    name: 'notifications',
    title: 'notifications',
    description: 'Event-driven email/SMS dispatch with template overrides.',
    overview:
      'The notifications module subscribes to domain events and dispatches email and SMS through pluggable providers. Templates can be overridden per tenant.',
    methods: [
      {
        name: 'Dispatch',
        signature: 'func (m *Module) Dispatch(ctx context.Context, n Notification) error',
        description: 'Renders a template and sends through the configured channel.',
      },
    ],
    events: [
      {
        name: 'auth.owner.signed_up',
        direction: 'Consumed',
        description: 'Triggers the welcome email.',
        payload: `{ "user_id", "tenant_id", "email" }`,
      },
    ],
  }),
  genericModule({
    slug: 'user',
    name: 'user',
    title: 'user',
    description: 'UserProfile lazy creation, GET/PATCH.',
    overview:
      'The user module stores extended profile data. Profiles are created lazily on first read so auth stays lightweight.',
    methods: [
      {
        name: 'GetProfile',
        signature:
          'func (m *Module) GetProfile(ctx context.Context, userID string) (UserProfile, error)',
        description: 'Returns the profile, creating an empty one on first access.',
      },
      {
        name: 'PatchProfile',
        signature:
          'func (m *Module) PatchProfile(ctx context.Context, userID string, p ProfilePatch) (UserProfile, error)',
        description: 'Applies a partial update to the profile.',
      },
    ],
    endpoints: [
      {
        method: 'GET',
        path: '/api/v1/users/me',
        auth: 'Bearer',
        responses: [{ status: '200', body: `{ "user_id", "display_name", "avatar_url" }` }],
      },
      {
        method: 'PATCH',
        path: '/api/v1/users/me',
        auth: 'Bearer',
        body: `{ "display_name", "avatar_url" }`,
        responses: [{ status: '200', body: `{ "user_id", "display_name" }` }],
      },
    ],
  }),
  genericModule({
    slug: 'storage',
    name: 'storage',
    title: 'storage',
    description: 'S3 backend (R2/MinIO/AWS/B2).',
    overview:
      'The storage module abstracts object storage over any S3-compatible backend, issuing presigned upload and download URLs scoped to a tenant.',
    methods: [
      {
        name: 'PresignUpload',
        signature:
          'func (m *Module) PresignUpload(ctx context.Context, in UploadRequest) (PresignedURL, error)',
        description: 'Returns a short-lived presigned PUT URL.',
      },
    ],
    config: [
      { name: 'DB', type: '*sql.DB', required: true, description: 'Database handle.' },
      { name: 'Endpoint', type: 'string', required: true, description: 'S3-compatible endpoint URL.' },
      { name: 'Bucket', type: 'string', required: true, description: 'Target bucket name.' },
    ],
  }),
  genericModule({
    slug: 'end-customers',
    name: 'endcustomers',
    title: 'end-customers',
    description: 'B2B2C CRUD + suspend/restore.',
    overview:
      'The end-customers module gives each of your tenants their own customers (B2B2C). It supports CRUD plus suspend and restore with full audit coverage.',
    methods: [
      {
        name: 'Suspend',
        signature: 'func (m *Module) Suspend(ctx context.Context, id string) error',
        description: 'Suspends an end-customer, blocking customer-auth logins.',
      },
      {
        name: 'Restore',
        signature: 'func (m *Module) Restore(ctx context.Context, id string) error',
        description: 'Restores a previously suspended end-customer.',
      },
    ],
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/end-customers',
        auth: 'Bearer (member)',
        body: `{ "email", "name" }`,
        responses: [{ status: '201', body: `{ "id", "email", "status": "active" }` }],
      },
    ],
  }),
  genericModule({
    slug: 'white-label',
    name: 'whitelabel',
    title: 'white-label',
    description: 'Per-tenant brand + custom domain DNS.',
    overview:
      'The white-label module lets each tenant set branding and attach a custom domain, verifying ownership through DNS records.',
    methods: [
      {
        name: 'AttachDomain',
        signature:
          'func (m *Module) AttachDomain(ctx context.Context, in DomainInput) (Domain, error)',
        description: 'Registers a custom domain and returns the DNS records to add.',
      },
    ],
  }),
  genericModule({
    slug: 'customer-auth',
    name: 'customerauth',
    title: 'customer-auth',
    description: 'Register/Login/OTP reset (aud:"customer").',
    overview:
      'The customer-auth module authenticates end-customers (not owners). Tokens carry aud:"customer" so they can never be used against owner endpoints.',
    methods: [
      {
        name: 'Register',
        signature: 'func (m *Module) Register(ctx context.Context, in RegisterInput) (Session, error)',
        description: 'Registers an end-customer under a tenant.',
      },
      {
        name: 'RequestOTP',
        signature: 'func (m *Module) RequestOTP(ctx context.Context, email string) error',
        description: 'Sends a one-time password for password reset.',
      },
    ],
    endpoints: [
      {
        method: 'POST',
        path: '/api/v1/customer/auth/login',
        auth: 'none',
        body: `{ "email", "password", "tenant_id" }`,
        responses: [{ status: '200', body: `{ "access_token", "aud": "customer" }` }],
      },
    ],
  }),
  genericModule({
    slug: 'customer-notifications',
    name: 'customernotifications',
    title: 'customer-notifications',
    description: 'Email+SMS, tenant overrides, event-driven.',
    overview:
      'The customer-notifications module mirrors notifications but targets end-customers, honoring per-tenant template and sender overrides.',
    events: [
      {
        name: 'customer.registered',
        direction: 'Consumed',
        description: 'Triggers the customer welcome message.',
        payload: `{ "customer_id", "tenant_id", "email" }`,
      },
    ],
  }),
]

export const modules: Record<string, ModuleDoc> = Object.fromEntries(
  [audit, auth, ...others].map((m) => [m.slug, m]),
)

export function getModule(slug: string): ModuleDoc | undefined {
  return modules[slug]
}
