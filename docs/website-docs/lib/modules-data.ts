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
  method: 'GET' | 'POST' | 'PUT' | 'PUT' | 'PATCH' | 'DELETE'
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

// ─── audit ────────────────────────────────────────────────────────────────────

const audit: ModuleDoc = {
  slug: 'audit',
  name: 'audit',
  title: 'audit',
  description:
    'Append-only event logging with v1 offset and v2 cursor pagination, plus at-least-once outbox delivery via the kernel/events bus.',
  overview:
    'The audit module records an immutable, append-only trail of everything that happens inside a tenant. Events are written through the transactional outbox (kernel/events) and delivered at-least-once to registered consumers. Reads support both v1 offset pagination (backward compatible) and the recommended v2 cursor pagination, which stays stable under concurrent writes. Other platforms depend on the EventLogger interface, not the concrete use case, so audit can evolve without breaking them.',
  install: `import "github.com/agile-factory/kernel/platform/audit"

// Include audit.Migrations in the slice passed to migrator.Run.
func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, audit.Migrations)
}`,
  config: [
    {
      name: 'DB',
      type: '*pgxpool.Pool',
      required: true,
      description: 'Database pool used by the outbox publisher for transactional writes.',
    },
    {
      name: 'Repo',
      type: 'domain.AuditRepository',
      required: true,
      description: 'Persistence port for audit log rows. Inject the Postgres implementation.',
    },
    {
      name: 'MembershipRepo',
      type: 'memberdomain.MembershipRepository',
      required: true,
      description: 'Consulted by HTTP handlers to gate list endpoints to tenant members.',
    },
    {
      name: 'Publisher',
      type: 'events.Publisher',
      required: true,
      description: 'Outbox publisher from kernel/events. Delivers audit.log_created at-least-once.',
    },
    {
      name: 'Logger',
      type: '*slog.Logger',
      required: true,
      description: 'Structured logger for internal diagnostics.',
    },
  ],
  usage: `auditMod, err := audit.New(audit.Config{
\tDB:             pool,
\tRepo:           auditinfra.NewPostgresAuditRepository(pool),
\tMembershipRepo: memberRepo,
\tPublisher:      bus.Publisher(),
\tLogger:         log,
})

// Register the consumer so audit.log_created events are persisted.
auditMod.Subscribe(bus)

// Expose the EventLogger interface to other modules.
authMod := auth.New(auth.Config{
\tAuditLogger: auditMod.EventLogger(),
\t// ...
})

auditMod.RegisterRoutes(mux)`,
  methods: [
    {
      name: 'New',
      signature: 'func New(cfg Config) (*Module, error)',
      description:
        'Wires the audit module. Returns an error listing all missing required Config fields — fail fast at startup rather than at runtime.',
    },
    {
      name: 'EventLogger',
      signature: 'func (m *Module) EventLogger() EventLogger',
      description:
        'Returns the audit producer contract (EventLogger interface). Wire this into the AuditLogger field of other modules so they emit audit events without importing concrete use cases.',
      example: `// In main.go
auditMod, _ := audit.New(cfg)
authMod := auth.New(auth.Config{
\tAuditLogger: auditMod.EventLogger(),
})`,
    },
    {
      name: 'Subscribe',
      signature: 'func (m *Module) Subscribe(sub events.Subscriber)',
      description:
        'Registers the audit consumer on the outbox bus. Call before bus.Run to ensure audit.log_created events are delivered and persisted with at-least-once guarantees.',
    },
    {
      name: 'RegisterRoutes',
      signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)',
      description:
        'Mounts all v1 and v2 audit HTTP endpoints on the provided mux. See HTTP Endpoints section for the full route table.',
    },
  ],
  events: [
    {
      name: 'audit.log_created',
      direction: 'Consumed',
      description:
        'Published by the LogEventUseCase into the outbox, then consumed by the module itself to persist the row in audit_logs with at-least-once delivery.',
      payload: `{
  "id":          "evt_01HF...",
  "tenant_id":   "tnt_01HF...",
  "actor_id":    "usr_01HF...",
  "action":      "tenant.settings.updated",
  "target":      "tnt_01HF...",
  "metadata":    { "field": "billing_email" },
  "recorded_at": "2026-06-29T10:12:04Z"
}`,
    },
  ],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/tenants/{tenantId}/audit-logs',
      auth: 'Bearer (member)',
      responses: [
        {
          status: '200',
          body: `{ "items": [...], "total": 42, "page": 1, "page_size": 20 }`,
        },
      ],
    },
    {
      method: 'GET',
      path: '/api/v1/audit-logs',
      auth: 'Bearer (admin)',
      responses: [
        {
          status: '200',
          body: `{ "items": [...], "total": 100, "page": 1, "page_size": 20 }`,
        },
      ],
    },
    {
      method: 'GET',
      path: '/api/v2/tenants/{tenantId}/audit-logs',
      auth: 'Bearer (member)',
      responses: [
        {
          status: '200',
          body: `{ "items": [...], "next_cursor": "c_8f2c..." }`,
        },
      ],
    },
    {
      method: 'GET',
      path: '/api/v2/audit-logs',
      auth: 'Bearer (admin)',
      responses: [
        {
          status: '200',
          body: `{ "items": [...], "next_cursor": "c_8f2c..." }`,
        },
      ],
    },
  ],
  migrations: [
    {
      file: '0001_create_audit_logs.up.sql',
      description: 'Append-only audit_logs table with tenant_id and recorded_at index.',
    },
    {
      file: '0002_add_audit_logs_cursor_indexes.up.sql',
      description: 'Composite (tenant_id, id) index backing v2 stable cursor pagination.',
    },
    {
      file: '0003_add_audit_logs_outbox_event_id.up.sql',
      description: 'outbox_event_id column for idempotent at-least-once delivery.',
    },
  ],
}

// ─── auth ─────────────────────────────────────────────────────────────────────

const auth: ModuleDoc = {
  slug: 'auth',
  name: 'auth',
  title: 'auth',
  description:
    'Owner signup, invite acceptance, email verification, login, refresh, and logout — JWT access tokens with rotating refresh tokens. Aligned with RFC-0018.',
  overview:
    'The auth module handles the full owner authentication lifecycle. Signup provisions the first owner and their tenant atomically in one transaction. Sessions use short-lived JWT access tokens paired with rotating, revocable refresh tokens (30-day TTL by default). Every sensitive action emits an audit event when an AuditLogger is wired in. The module intentionally has no /register endpoint — B2B onboarding is invite-only; new workspace users always arrive via accept-invite.',
  install: `import "github.com/agile-factory/kernel/platform/auth"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool,
\t\tauth.Migrations,
\t\t// compose with other modules...
\t)
}`,
  config: [
    {
      name: 'Pool',
      type: '*pgxpool.Pool',
      required: true,
      description: 'Database pool for use cases that wrap work in a transaction (SignupOwner, AcceptInvite, VerifyEmail, Refresh).',
    },
    {
      name: 'UserRepo',
      type: 'domain.UserRepository',
      required: true,
      description: 'Persistence port for auth.User records.',
    },
    {
      name: 'TenantRepo',
      type: 'tenantdomain.TenantRepository',
      required: true,
      description: 'Used by SignupOwner to create the first tenant atomically.',
    },
    {
      name: 'TenantSettingsRepo',
      type: 'tenantdomain.TenantSettingsRepository',
      required: true,
      description: 'Seeds default settings alongside the new tenant.',
    },
    {
      name: 'MembershipRepo',
      type: 'memberdomain.MembershipRepository',
      required: true,
      description: 'Creates the OWNER membership row during signup.',
    },
    {
      name: 'InviteTokenRepo',
      type: 'memberdomain.InviteTokenRepository',
      required: true,
      description: 'Validates and consumes invite tokens on AcceptInvite.',
    },
    {
      name: 'RefreshTokenRepo',
      type: 'domain.RefreshTokenRepository',
      required: true,
      description: 'Stores rotating refresh tokens for session management.',
    },
    {
      name: 'EmailVerifRepo',
      type: 'domain.EmailVerificationTokenRepository',
      required: true,
      description: 'Issues and validates one-time email verification tokens.',
    },
    {
      name: 'MembershipListUC',
      type: '*memberusecase.ListMembersUseCase',
      required: false,
      description: 'Injected so the auth handler can return membership context on login.',
    },
    {
      name: 'AuditLogger',
      type: 'audit.EventLogger',
      required: false,
      description: 'When set, auth actions (signup, login, logout) are recorded to the audit trail.',
    },
    {
      name: 'Publisher',
      type: 'events.Publisher',
      required: false,
      description: 'Outbox publisher. When nil, auth.signup_owner and auth.invite_accepted events are silently skipped.',
    },
    {
      name: 'Logger',
      type: '*slog.Logger',
      required: true,
      description: 'Structured logger.',
    },
    {
      name: 'JWTSecret',
      type: 'string',
      required: true,
      description: 'HMAC-SHA256 secret used to sign and verify access tokens.',
    },
    {
      name: 'TokenLifespan',
      type: 'time.Duration',
      required: false,
      description: 'Access token lifetime. Defaults to 15 minutes.',
    },
    {
      name: 'RefreshTokenTTL',
      type: 'time.Duration',
      required: false,
      description: 'Refresh token lifetime. Defaults to 30 days.',
    },
    {
      name: 'BCryptCost',
      type: 'int',
      required: false,
      description: 'bcrypt cost factor. 0 uses the bcrypt library default (currently 10).',
    },
  ],
  usage: `authMod := auth.New(auth.Config{
\tPool:             pool,
\tUserRepo:         authinfra.NewPostgresUserRepository(pool),
\tTenantRepo:       tenantRepo,
\tTenantSettingsRepo: settingsRepo,
\tMembershipRepo:   memberRepo,
\tInviteTokenRepo:  inviteTokenRepo,
\tRefreshTokenRepo: authinfra.NewPostgresRefreshTokenRepository(pool),
\tEmailVerifRepo:   authinfra.NewPostgresEmailVerifRepository(pool),
\tAuditLogger:      auditMod.EventLogger(),
\tPublisher:        bus.Publisher(),
\tLogger:           log,
\tJWTSecret:        os.Getenv("JWT_SECRET"),
\tTokenLifespan:    15 * time.Minute,
})

authMod.RegisterRoutes(mux)

// Expose JWTService to other modules that need to validate tokens.
jwtSvc := authMod.JWTService()`,
  methods: [
    {
      name: 'New',
      signature: 'func New(cfg Config) *Module',
      description:
        'Wires all internal components: hasher, JWT service, and all six use cases. Applies default values for RefreshTokenTTL (30 days).',
    },
    {
      name: 'JWTService',
      signature: 'func (m *Module) JWTService() *infrastructure.JWTService',
      description:
        'Returns the JWT signer/validator for middlewares that need to verify tokens issued by this module. Inject into other modules (e.g. customer_auth) rather than re-creating a service.',
      example: `// In middleware
jwtSvc := authMod.JWTService()
mux.Handle("/api/v1/tenants/", authMiddleware(jwtSvc, tenantMod.Handler()))`,
    },
    {
      name: 'RegisterRoutes',
      signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)',
      description:
        'Mounts POST /api/v1/auth/* endpoints on the provided mux.',
    },
  ],
  events: [
    {
      name: 'auth.signup_owner',
      direction: 'Published',
      description: 'Emitted when a new owner and tenant are provisioned atomically. Consumed by the notification module (email verification) and the user module (profile creation).',
      payload: `{
  "user_id":            "usr_01HF...",
  "tenant_id":          "tnt_01HF...",
  "email":              "owner@acme.io",
  "tenant_name":        "Acme Corp",
  "verification_token": "evtok_01HF..."
}`,
    },
    {
      name: 'auth.invite_accepted',
      direction: 'Published',
      description: 'Emitted when an invitee successfully consumes an invite token. Consumed by the notification module to notify the inviter.',
      payload: `{
  "invitee_name":  "Alice",
  "invitee_email": "alice@acme.io",
  "inviter_id":    "usr_01HF...",
  "inviter_email": "bob@acme.io",
  "inviter_name":  "Bob",
  "role":          "ADMIN",
  "tenant_name":   "Acme Corp"
}`,
    },
  ],
  endpoints: [
    {
      method: 'POST',
      path: '/api/v1/auth/signup-owner',
      auth: 'none',
      body: `{ "email": "owner@acme.io", "password": "s3cr3t", "tenant_name": "Acme Corp" }`,
      responses: [
        { status: '201', body: `{ "access_token", "refresh_token", "user": { "id", "email", "email_verified" } }` },
        { status: '409', body: `{ "error": "email_taken" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/accept-invite',
      auth: 'none',
      body: `{ "token": "invtok_01HF...", "password": "s3cr3t" }`,
      responses: [
        { status: '201', body: `{ "access_token", "refresh_token", "user" }` },
        { status: '400', body: `{ "error": "invalid_or_expired_token" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/verify-email',
      auth: 'none',
      body: `{ "token": "evtok_01HF..." }`,
      responses: [
        { status: '204', body: `(no content)` },
        { status: '400', body: `{ "error": "invalid_or_expired_token" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/login',
      auth: 'none',
      body: `{ "email": "owner@acme.io", "password": "s3cr3t" }`,
      responses: [
        { status: '200', body: `{ "access_token", "refresh_token", "user" }` },
        { status: '401', body: `{ "error": "invalid_credentials" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/refresh',
      auth: 'none',
      body: `{ "refresh_token": "rftok_01HF..." }`,
      responses: [
        { status: '200', body: `{ "access_token", "refresh_token" }` },
        { status: '401', body: `{ "error": "invalid_or_revoked_token" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/auth/logout',
      auth: 'Bearer',
      body: `{ "refresh_token": "rftok_01HF..." }`,
      responses: [
        { status: '204', body: `(no content)` },
      ],
    },
  ],
  migrations: [
    { file: '0001_create_users.up.sql', description: 'Users table with hashed credentials and email_verified flag.' },
    { file: '0002_drop_users_tenant_id.up.sql', description: 'Removes tenant_id from users — tenant membership is tracked in the membership module.' },
    { file: '0003_add_email_verification_to_users.up.sql', description: 'email_verified boolean on users table.' },
    { file: '0004_create_email_verification_tokens.up.sql', description: 'One-time email verification and invite redemption tokens.' },
    { file: '0005_create_refresh_tokens.up.sql', description: 'Rotating refresh-token store with per-row revocation.' },
  ],
}

// ─── tenant ───────────────────────────────────────────────────────────────────

const tenant: ModuleDoc = {
  slug: 'tenant',
  name: 'tenant',
  title: 'tenant',
  description: 'Tenant CRUD, per-tenant settings, and super-admin provisioning. The top-level isolation boundary every other module scopes to.',
  overview:
    'The tenant module manages the lifecycle of tenants and their settings. Tenants are the top-level isolation unit; every other module scopes its data by tenant_id. Public creation is intentionally absent — the only entry points are signup-owner (auth) and POST /admin/v1/tenants (super-admin sales-assisted onboarding). PATCH and DELETE are gated to OWNER/ADMIN roles via MembershipRepo.',
  install: `import "github.com/agile-factory/kernel/platform/tenant"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, tenant.Migrations)
}`,
  config: [
    { name: 'Pool', type: '*pgxpool.Pool', required: true, description: 'Database pool for transactional tenant creation.' },
    { name: 'Repo', type: 'domain.TenantRepository', required: true, description: 'Persistence port for Tenant records.' },
    { name: 'SettingsRepo', type: 'domain.TenantSettingsRepository', required: true, description: 'Persistence port for per-tenant settings.' },
    { name: 'MembershipRepo', type: 'memberdomain.MembershipRepository', required: true, description: 'Consulted by HTTP handlers for OWNER/ADMIN gating on PATCH/DELETE/settings endpoints.' },
    { name: 'AuditLogger', type: 'audit.EventLogger', required: false, description: 'When set, tenant mutations are recorded to the audit trail.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `tenantMod := tenant.New(tenant.Config{
\tPool:           pool,
\tRepo:           tenantinfra.NewPostgresTenantRepository(pool),
\tSettingsRepo:   tenantinfra.NewPostgresTenantSettingsRepository(pool),
\tMembershipRepo: memberRepo,
\tAuditLogger:    auditMod.EventLogger(),
\tLogger:         log,
})
tenantMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) *Module', description: 'Wires all six use cases and the HTTP handler.' },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts all tenant endpoints (see HTTP Endpoints).' },
    { name: 'CreateUC', signature: '*usecase.CreateTenantUseCase', description: 'Creates a tenant and seeds default settings in one transaction. Exposed on the Module for platform-level composition (e.g. auth.SignupOwner).' },
    { name: 'FindUC', signature: '*usecase.FindTenantUseCase', description: 'Fetches a tenant by ID or slug. Exported for use by other modules needing to look up tenant context.' },
    { name: 'UpdateSettingsUC', signature: '*usecase.UpdateTenantSettingsUseCase', description: 'Patches per-tenant settings. Emits an audit event when AuditLogger is wired.' },
  ],
  events: [],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/tenants/{id}',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "id", "name", "slug", "created_at" }` }, { status: '404', body: `{ "error": "tenant_not_found" }` }],
    },
    {
      method: 'GET',
      path: '/api/v1/tenants/by-slug',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "id", "name", "slug" }` }],
    },
    {
      method: 'PATCH',
      path: '/api/v1/tenants/{id}',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "name": "Acme Corp" }`,
      responses: [{ status: '200', body: `{ "id", "name", "slug" }` }],
    },
    {
      method: 'DELETE',
      path: '/api/v1/tenants/{id}',
      auth: 'Bearer (OWNER)',
      responses: [{ status: '204', body: `(no content — soft delete)` }],
    },
    {
      method: 'GET',
      path: '/api/v1/tenants/{id}/settings',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "tenant_id", "settings": { ... } }` }],
    },
    {
      method: 'PATCH',
      path: '/api/v1/tenants/{id}/settings',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "settings": { "billing_email": "billing@acme.io" } }`,
      responses: [{ status: '200', body: `{ "tenant_id", "settings" }` }],
    },
    {
      method: 'POST',
      path: '/admin/v1/tenants',
      auth: 'Super-admin key',
      body: `{ "name": "New Corp", "owner_email": "ceo@newcorp.io" }`,
      responses: [{ status: '201', body: `{ "id", "name", "slug" }` }],
    },
  ],
  migrations: [
    { file: '0001_create_tenants.up.sql', description: 'Tenants table with slug, status, and soft-delete support.' },
    { file: '0002_create_tenant_settings.up.sql', description: 'JSONB tenant_settings table keyed by tenant_id.' },
  ],
}

// ─── membership ───────────────────────────────────────────────────────────────

const membership: ModuleDoc = {
  slug: 'membership',
  name: 'membership',
  title: 'membership',
  description: 'Invite, revoke, and list members of a tenant. Tokenized invite flow consumed by auth.AcceptInvite. RFC-0018 §6.',
  overview:
    'The membership module governs who belongs to a tenant and in what role. Invites are tokenized — an invite_token row is created and emailed to the invitee; the auth module consumes the token via AcceptInvite, creating the user and an ACTIVE membership atomically. There is no PENDING membership state; the token IS the pending state. A partial unique index prevents duplicate active memberships for the same user/tenant pair.',
  install: `import "github.com/agile-factory/kernel/platform/membership"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, membership.Migrations)
}`,
  config: [
    { name: 'Pool', type: '*pgxpool.Pool', required: true, description: 'Database pool for transactional invite creation.' },
    { name: 'Repo', type: 'domain.MembershipRepository', required: true, description: 'Persistence port for Membership records.' },
    { name: 'InviteTokenRepo', type: 'domain.InviteTokenRepository', required: true, description: 'Persistence port for invite tokens.' },
    { name: 'TenantRepo', type: 'tenantdomain.TenantRepository', required: true, description: 'Validates that the target tenant exists before issuing an invite.' },
    { name: 'UserLookup', type: 'usecase.UserLookup', required: true, description: 'Interface for looking up a user by email — typically the auth UserRepo wrapped to satisfy this interface.' },
    { name: 'AuditLogger', type: 'audit.EventLogger', required: false, description: 'Records invite and revoke actions to the audit trail.' },
    { name: 'Publisher', type: 'events.Publisher', required: false, description: 'When nil, the member.invited event is skipped.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `memberMod := membership.New(membership.Config{
\tPool:            pool,
\tRepo:            memberinfra.NewPostgresMembershipRepository(pool),
\tInviteTokenRepo: memberinfra.NewPostgresInviteTokenRepository(pool),
\tTenantRepo:      tenantRepo,
\tUserLookup:      userRepo,
\tAuditLogger:     auditMod.EventLogger(),
\tPublisher:       bus.Publisher(),
\tLogger:          log,
})
memberMod.RegisterRoutes(mux)`,
  methods: [
    {
      name: 'InviteUC',
      signature: '*usecase.InviteMemberUseCase',
      description: 'Creates an invite_token and publishes member.invited. Exported so auth.New can inject it for listing memberships on login.',
    },
    {
      name: 'RevokeUC',
      signature: '*usecase.RevokeMemberUseCase',
      description: 'Removes a membership row and emits an audit event.',
    },
    {
      name: 'ListUC',
      signature: '*usecase.ListMembersUseCase',
      description: 'Lists members by tenant or by user with cursor pagination. Exported for auth and tenant modules.',
    },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts all membership endpoints.' },
  ],
  events: [
    {
      name: 'member.invited',
      direction: 'Published',
      description: 'Emitted when an invite token is created. Consumed by the notification module to send the invitation email with the accept link.',
      payload: `{
  "email":        "alice@acme.io",
  "token":        "invtok_01HF...",
  "role":         "ADMIN",
  "tenant_name":  "Acme Corp",
  "inviter_name": "Bob",
  "expires_at":   "2026-07-29T10:00:00Z"
}`,
    },
  ],
  endpoints: [
    {
      method: 'POST',
      path: '/api/v1/memberships/invite',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "email": "alice@acme.io", "role": "ADMIN", "tenant_id": "tnt_01HF..." }`,
      responses: [
        { status: '201', body: `{ "invite_id", "email", "role", "expires_at" }` },
        { status: '409', body: `{ "error": "already_a_member" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/memberships/{id}/revoke',
      auth: 'Bearer (OWNER/ADMIN)',
      responses: [{ status: '204', body: `(no content)` }],
    },
    {
      method: 'GET',
      path: '/api/v1/memberships/tenant/{tenantId}',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "items": [{ "id", "user_id", "role", "status", "joined_at" }], "next_cursor" }` }],
    },
    {
      method: 'GET',
      path: '/api/v1/memberships/user/{userId}',
      auth: 'Bearer (self)',
      responses: [{ status: '200', body: `{ "items": [{ "tenant_id", "role", "status" }], "next_cursor" }` }],
    },
  ],
  migrations: [
    { file: '0001_create_memberships.up.sql', description: 'Memberships table with role and status columns.' },
    { file: '0002_add_owner_role.up.sql', description: 'Adds OWNER to the role enum.' },
    { file: '0003_create_invite_tokens.up.sql', description: 'Tokenized invite table replacing the PENDING membership pattern.' },
    { file: '0004_partial_unique_active_memberships.up.sql', description: 'Partial unique index on (user_id, tenant_id) WHERE status = ACTIVE — prevents duplicate memberships.' },
  ],
}

// ─── billing ──────────────────────────────────────────────────────────────────

const billing: ModuleDoc = {
  slug: 'billing',
  name: 'billing',
  title: 'billing',
  description: 'Plan listing, subscription lookup, Stripe webhook reconciliation, and Wave stub for African payment rails.',
  overview:
    'The billing module manages plans and subscriptions and reconciles state from payment providers. Stripe is supported via signed webhook verification. A Wave stub is provided for West African payment rails (Wave Money). The module exposes ListPlans and GetSubscription use cases via HTTP, and handles provider webhooks that update subscription state. It does not create subscriptions directly — subscription creation is initiated externally (Stripe Checkout, Wave redirect) and confirmed via webhook.',
  install: `import "github.com/agile-factory/kernel/platform/billing"
import "github.com/agile-factory/kernel/platform/billing/adapter"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, billing.Migrations)
}`,
  config: [
    { name: 'SubRepo', type: 'domain.SubscriptionRepository', required: true, description: 'Persistence port for Subscription records.' },
    { name: 'PlanRepo', type: 'domain.PlanRepository', required: true, description: 'Persistence port for Plan records (typically seeded at boot).' },
    {
      name: 'Processor',
      type: 'domain.PaymentProcessor',
      required: true,
      description: 'Payment provider adapter. Use adapter.NewStripeAdapter(...) for Stripe or the Wave stub.',
    },
    { name: 'Checker', type: 'permission.Checker', required: true, description: 'Permission checker for OWNER-gated subscription endpoints.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `stripeAdapter := adapter.NewStripeAdapter(adapter.StripeConfig{
\tSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
\tWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
})

billingMod, err := billing.New(billing.Config{
\tSubRepo:   billinginfra.NewPostgresSubscriptionRepository(pool),
\tPlanRepo:  billinginfra.NewPostgresPlanRepository(pool),
\tProcessor: stripeAdapter,
\tChecker:   permission.New(memberRepo),
\tLogger:    log,
})
billingMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires the billing module. Returns an error if any required Config field is nil.' },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts plans, subscription, and webhook endpoints.' },
  ],
  events: [
    {
      name: 'billing.invoice_paid',
      direction: 'Published',
      description: 'Published via webhook reconciliation when Stripe/Wave confirms a payment. Consumed by customer_notifications for invoice receipts.',
      payload: `{ "tenant_id", "amount": "49.00", "currency": "USD" }`,
    },
    {
      name: 'billing.payment_reminder',
      direction: 'Published',
      description: 'Published when a payment is past due. Consumed by customer_notifications.',
      payload: `{ "tenant_id", "amount": "49.00", "currency": "USD" }`,
    },
  ],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/plans',
      auth: 'Bearer',
      responses: [{ status: '200', body: `{ "items": [{ "id", "name", "price", "currency", "interval" }] }` }],
    },
    {
      method: 'GET',
      path: '/api/v1/tenants/{tenantId}/subscription',
      auth: 'Bearer (OWNER)',
      responses: [
        { status: '200', body: `{ "id", "plan_id", "status", "current_period_end" }` },
        { status: '404', body: `{ "error": "no_active_subscription" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/billing/webhooks/stripe',
      auth: 'Stripe-Signature header',
      responses: [
        { status: '200', body: `{ "received": true }` },
        { status: '400', body: `{ "error": "invalid_signature" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/billing/webhooks/wave',
      auth: 'Wave HMAC header',
      responses: [{ status: '200', body: `{ "received": true }` }],
    },
  ],
  migrations: [
    { file: '0001_create_billing_tables.up.sql', description: 'plans and subscriptions tables with provider_id and status columns.' },
  ],
}

// ─── notifications ────────────────────────────────────────────────────────────

const notifications: ModuleDoc = {
  slug: 'notifications',
  name: 'notification',
  title: 'notifications',
  description: 'Event-driven email dispatch (RFC-0019). Subscribes to outbox events, renders Go html/template templates, and delivers via a pluggable EmailSender.',
  overview:
    'The notifications module (package notification) subscribes to domain events from the outbox bus and dispatches transactional emails to workspace members and owners. Templates are parsed from embed.FS at boot — any syntax error surfaces at startup, not at runtime (Lucas amendment). Every delivery attempt is logged in notification_logs for operator audit and replay. No HTTP routes are exposed in v0.1 — an admin view over notification_logs is planned. SMS support is planned for a future RFC.',
  install: `import "github.com/agile-factory/kernel/platform/notification"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, notification.Migrations)
}`,
  config: [
    { name: 'LogRepo', type: 'domain.NotificationLogRepository', required: true, description: 'Persistence port for notification_logs — append-only delivery audit.' },
    { name: 'EmailSender', type: 'EmailSender', required: true, description: 'Email delivery adapter. Implement notification.EmailSender and inject your provider (SendGrid, Resend, SMTP, etc.).' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
    { name: 'From', type: 'string', required: true, description: 'From: header on outgoing emails (e.g. "Acme <noreply@acme.io>").' },
    { name: 'BaseURL', type: 'string', required: true, description: 'Public URL of the SaaS app — used to compose verify and accept links inside templates.' },
    { name: 'DefaultLocale', type: 'string', required: false, description: 'Locale for template rendering. Defaults to "en".' },
  ],
  usage: `notifMod, err := notification.New(notification.Config{
\tLogRepo:     notifinfra.NewPostgresNotificationLogRepository(pool),
\tEmailSender: mySendGridAdapter,
\tLogger:      log,
\tFrom:        "Acme <noreply@acme.io>",
\tBaseURL:     "https://app.acme.io",
})

// Register handlers before starting the bus.
notifMod.Subscribe(bus)
go bus.Run(ctx)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Builds the module. Parses all embedded email templates eagerly so template errors surface at startup.' },
    {
      name: 'Subscribe',
      signature: 'func (m *Module) Subscribe(bus events.Subscriber)',
      description: 'Wires three outbox handlers: auth.signup_owner → email verification, member.invited → invitation email, auth.invite_accepted → notify inviter.',
    },
    {
      name: 'Dispatch',
      signature: 'func (m *Module) Dispatch(ctx context.Context, event Event) error',
      description: 'Sends one notification directly, bypassing the outbox. Useful for scheduled jobs or tests.',
      example: `err := notifMod.Dispatch(ctx, notification.Event{
\tType:    notification.EventEmailVerificationRequested,
\tEmail:   "user@acme.io",
\tLocale:  "fr",
\tPayload: map[string]any{
\t\t"TenantName": "Acme",
\t\t"VerifyURL":  "https://app.acme.io/verify?token=...",
\t},
})`,
    },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(_ *http.ServeMux)', description: 'No-op in v0.1.0. Admin views over notification_logs are planned for a future release.' },
  ],
  events: [
    {
      name: 'auth.signup_owner',
      direction: 'Consumed',
      description: 'Triggers the email verification email with the verify URL. Requires email and verification_token in the payload.',
      payload: `{ "email", "tenant_name", "verification_token" }`,
    },
    {
      name: 'member.invited',
      direction: 'Consumed',
      description: 'Triggers the member invitation email with the accept-invite URL, role, and expiry.',
      payload: `{ "email", "token", "role", "tenant_name", "inviter_name", "expires_at" }`,
    },
    {
      name: 'auth.invite_accepted',
      direction: 'Consumed',
      description: 'Notifies the inviter that their invitation was accepted. No-op stub in v0.1 — template ships in v0.2.',
      payload: `{ "invitee_name", "invitee_email", "inviter_email", "inviter_name", "role", "tenant_name" }`,
    },
  ],
  endpoints: [],
  migrations: [
    { file: '0001_create_notification_logs.up.sql', description: 'Append-only notification_logs table tracking status (pending / sent / failed) per attempt.' },
  ],
}

// ─── user ─────────────────────────────────────────────────────────────────────

const user: ModuleDoc = {
  slug: 'user',
  name: 'user',
  title: 'user',
  description: 'User profile (display name, avatar, locale, timezone) separate from auth identity. Lazy creation via outbox. RFC-0009.',
  overview:
    'The user module manages UserProfile records separately from auth.User (identity + credentials), following RFC-0009 single-responsibility separation. Profiles are created lazily on signup via the auth.signup_owner outbox event — the owner always has a profile row from the moment they sign up, so GET /profile never returns 404. Members who joined via invite get a profile created lazily on first GET.',
  install: `import "github.com/agile-factory/kernel/platform/user"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, user.Migrations)
}`,
  config: [
    { name: 'Repo', type: 'domain.UserProfileRepository', required: true, description: 'Persistence port for UserProfile records.' },
    { name: 'Checker', type: 'permission.Checker', required: true, description: 'Permission checker for self-only access enforcement on profile endpoints.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `userMod, err := user.New(user.Config{
\tRepo:    userinfra.NewPostgresUserProfileRepository(pool),
\tChecker: permission.New(memberRepo),
\tLogger:  log,
})

// Creates the owner's profile on signup automatically.
userMod.Subscribe(bus)

userMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires the get and update profile use cases. Returns an error if any required field is nil.' },
    {
      name: 'Subscribe',
      signature: 'func (m *Module) Subscribe(bus events.Subscriber)',
      description: 'Registers a handler for auth.signup_owner that creates the owner\'s UserProfile with an email-derived DisplayName at signup time.',
    },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts GET and PATCH /api/v1/users/{userId}/profile.' },
  ],
  events: [
    {
      name: 'auth.signup_owner',
      direction: 'Consumed',
      description: 'Triggers profile creation for the new owner. DisplayName is derived from the email prefix. Failure is logged but does not retry — profile creation is best-effort.',
      payload: `{ "user_id", "email" }`,
    },
  ],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/users/{userId}/profile',
      auth: 'Bearer (self)',
      responses: [
        { status: '200', body: `{ "user_id", "display_name", "avatar_url", "locale", "timezone" }` },
      ],
    },
    {
      method: 'PATCH',
      path: '/api/v1/users/{userId}/profile',
      auth: 'Bearer (self)',
      body: `{ "display_name": "Alice", "avatar_url": "https://...", "locale": "fr", "timezone": "Africa/Dakar" }`,
      responses: [
        { status: '200', body: `{ "user_id", "display_name", "avatar_url", "locale", "timezone" }` },
      ],
    },
  ],
  migrations: [
    { file: '0001_create_user_profiles.up.sql', description: 'user_profiles table with display_name, avatar_url, locale, and timezone columns.' },
  ],
}

// ─── storage ──────────────────────────────────────────────────────────────────

const storage: ModuleDoc = {
  slug: 'storage',
  name: 'storage',
  title: 'storage',
  description: 'S3-compatible object storage wrapper (Cloudflare R2, MinIO, AWS S3, B2). No HTTP routes or migrations in v0.1. RFC-0017 + RFC-0038.',
  overview:
    'The storage module packages an S3-compatible backend so a consumer can wire it in one line and inject the resulting storage.Storage interface into any module that needs to read or write blobs (logos, exports, attachments). The module has no HTTP routes and no database tables in v0.1.0 — RegisterRoutes is a no-op. Presigned URL HTTP endpoints are planned for RFC-0017 phase 2. New() validates the backend is reachable at boot (BucketExists round-trip) so misconfigured credentials fail fast instead of causing runtime 5xx errors.',
  install: `import "github.com/agile-factory/kernel/platform/storage"

// storage has no SQL migrations — omit it from migrator.Run.`,
  config: [
    {
      name: 'S3',
      type: 'S3Config',
      required: true,
      description: 'S3-compatible backend config: Endpoint, Bucket, Region, AccessKeyID, SecretAccessKey.',
    },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger. Used to confirm backend readiness at startup.' },
  ],
  usage: `storageMod, err := storage.New(ctx, storage.Config{
\tS3: storage.S3Config{
\t\tEndpoint:        "https://<account>.r2.cloudflarestorage.com",
\t\tBucket:          "acme-uploads",
\t\tRegion:          "auto",
\t\tAccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
\t\tSecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
\t},
\tLogger: log,
})

// Inject the Storage interface into modules that need blob I/O.
backend := storageMod.Storage()`,
  methods: [
    {
      name: 'New',
      signature: 'func New(ctx context.Context, cfg Config) (*Module, error)',
      description: 'Constructs the module and performs a BucketExists round-trip to validate credentials and bucket name at startup.',
    },
    {
      name: 'Storage',
      signature: 'func (m *Module) Storage() Storage',
      description: 'Returns the storage.Storage interface. Inject this into any module that needs Put / Get / Delete / List operations on blobs.',
    },
    {
      name: 'RegisterRoutes',
      signature: 'func (m *Module) RegisterRoutes(_ *http.ServeMux)',
      description: 'No-op in v0.1.0. Presigned URL endpoints are planned for RFC-0038 phase 2.',
    },
  ],
  events: [],
  endpoints: [],
  migrations: [],
}

// ─── end-customers ────────────────────────────────────────────────────────────

const endCustomers: ModuleDoc = {
  slug: 'end-customers',
  name: 'end_customers',
  title: 'end-customers',
  description: 'B2B2C end-customer CRUD with suspend/restore and full audit coverage. RFC-0033.',
  overview:
    'The end-customers module manages end-customers — the consumers of a tenant\'s customer-facing portal. Distinct from auth.User (workspace members). Each end-customer belongs to one tenant. All write operations publish outbox events consumed by customer_notifications (welcome email, etc.). Suspend blocks customer-auth logins without deleting data; Restore re-enables them.',
  install: `import "github.com/agile-factory/kernel/platform/end_customers"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, end_customers.Migrations)
}`,
  config: [
    { name: 'DB', type: '*pgxpool.Pool', required: false, description: 'Database pool for transactional writes. May be nil in unit tests when repositories are stubs.' },
    { name: 'Repo', type: 'domain.EndCustomerRepository', required: true, description: 'Persistence port for EndCustomer records.' },
    { name: 'Checker', type: 'permission.Checker', required: true, description: 'Permission checker — all endpoints require an active tenant membership.' },
    { name: 'Publisher', type: 'events.Publisher', required: true, description: 'Outbox publisher for end_customer.created and lifecycle events.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `ecMod, err := end_customers.New(end_customers.Config{
\tDB:        pool,
\tRepo:      ecinfra.NewPostgresEndCustomerRepository(pool),
\tChecker:   permission.New(memberRepo),
\tPublisher: bus.Publisher(),
\tLogger:    log,
})
ecMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires seven use cases: Create, Get, Update, Delete, Suspend, Restore, List.' },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts all seven end-customer endpoints.' },
  ],
  events: [
    {
      name: 'end_customer.created',
      direction: 'Published',
      description: 'Emitted when a new end-customer is created. Consumed by customer_notifications to send a welcome message.',
      payload: `{ "end_customer_id", "tenant_id", "email", "name" }`,
    },
  ],
  endpoints: [
    {
      method: 'POST',
      path: '/api/v1/end-customers',
      auth: 'Bearer (member)',
      body: `{ "email": "customer@user.io", "name": "Jane Doe" }`,
      responses: [{ status: '201', body: `{ "id", "tenant_id", "email", "name", "status": "active" }` }],
    },
    {
      method: 'GET',
      path: '/api/v1/end-customers',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "items": [...], "next_cursor" }` }],
    },
    {
      method: 'GET',
      path: '/api/v1/end-customers/{id}',
      auth: 'Bearer (member)',
      responses: [
        { status: '200', body: `{ "id", "tenant_id", "email", "name", "status", "created_at" }` },
        { status: '404', body: `{ "error": "end_customer_not_found" }` },
      ],
    },
    {
      method: 'PATCH',
      path: '/api/v1/end-customers/{id}',
      auth: 'Bearer (member)',
      body: `{ "name": "Jane Smith" }`,
      responses: [{ status: '200', body: `{ "id", "name", "status" }` }],
    },
    {
      method: 'DELETE',
      path: '/api/v1/end-customers/{id}',
      auth: 'Bearer (member)',
      responses: [{ status: '204', body: `(no content)` }],
    },
    {
      method: 'POST',
      path: '/api/v1/end-customers/{id}/suspend',
      auth: 'Bearer (member)',
      responses: [{ status: '204', body: `(no content — blocks customer-auth logins)` }],
    },
    {
      method: 'POST',
      path: '/api/v1/end-customers/{id}/restore',
      auth: 'Bearer (member)',
      responses: [{ status: '204', body: `(no content — re-enables customer-auth logins)` }],
    },
  ],
  migrations: [
    { file: '0001_create_end_customers.up.sql', description: 'end_customers table with tenant_id, email, name, and status columns.' },
    { file: '0002_create_end_customer_tags.up.sql', description: 'JSONB tags column for free-form tenant-defined metadata.' },
  ],
}

// ─── white-label ──────────────────────────────────────────────────────────────

const whiteLabel: ModuleDoc = {
  slug: 'white-label',
  name: 'white_label',
  title: 'white-label',
  description: 'Per-tenant branding (logo, colors, custom domain), DNS verification, and email template overrides for the B2B2C portal. RFC-0035.',
  overview:
    'The white-label module lets each tenant set visual branding and attach a custom domain for their customer-facing portal. Custom domain verification requires the tenant to add a DNS TXT record; the module\'s VerifyDomain use case resolves it via the standard library (or an injected DNSResolver in tests). A public endpoint returns brand settings without authentication so the customer portal can render the correct theme before login. All mutations publish outbox events for audit coverage.',
  install: `import "github.com/agile-factory/kernel/platform/white_label"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, white_label.Migrations)
}`,
  config: [
    { name: 'DB', type: '*pgxpool.Pool', required: false, description: 'Database pool for transactional writes. May be nil in unit tests.' },
    { name: 'Repo', type: 'domain.BrandRepository', required: true, description: 'Persistence port for BrandSettings records.' },
    { name: 'Checker', type: 'permission.Checker', required: true, description: 'Permission checker — brand mutations require OWNER/ADMIN role.' },
    { name: 'Publisher', type: 'events.Publisher', required: true, description: 'Outbox publisher for brand and domain lifecycle events.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
    { name: 'DNSResolver', type: 'usecase.DNSResolver', required: false, description: 'DNS resolver used by VerifyDomain. Defaults to stdlib net.DefaultResolver. Override in tests for deterministic verification.' },
  ],
  usage: `wlMod, err := white_label.New(white_label.Config{
\tDB:        pool,
\tRepo:      wlinfra.NewPostgresBrandRepository(pool),
\tChecker:   permission.New(memberRepo),
\tPublisher: bus.Publisher(),
\tLogger:    log,
})
wlMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires four use cases: GetBrand, UpdateBrand, VerifyDomain, GetPublicBrand.' },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts brand and domain endpoints on both /api/v1 and /public/v1.' },
  ],
  events: [],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/tenants/{id}/brand',
      auth: 'Bearer (member)',
      responses: [{ status: '200', body: `{ "tenant_id", "logo_url", "primary_color", "custom_domain", "domain_verified" }` }],
    },
    {
      method: 'PATCH',
      path: '/api/v1/tenants/{id}/brand',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "logo_url": "https://cdn.acme.io/logo.svg", "primary_color": "#0055FF" }`,
      responses: [{ status: '200', body: `{ "tenant_id", "logo_url", "primary_color" }` }],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/brand/verify-domain',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "domain": "portal.acme.io" }`,
      responses: [
        { status: '200', body: `{ "domain", "domain_verified": true }` },
        { status: '422', body: `{ "error": "dns_record_not_found" }` },
      ],
    },
    {
      method: 'GET',
      path: '/public/v1/brand',
      auth: 'none (by domain header)',
      responses: [{ status: '200', body: `{ "logo_url", "primary_color", "tenant_name" }` }],
    },
  ],
  migrations: [
    { file: '0001_create_brand_settings.up.sql', description: 'brand_settings table with logo_url, primary_color, custom_domain, and domain_verified columns.' },
  ],
}

// ─── customer-auth ────────────────────────────────────────────────────────────

const customerAuth: ModuleDoc = {
  slug: 'customer-auth',
  name: 'customer_auth',
  title: 'customer-auth',
  description: 'End-customer register/login/refresh/logout and password reset. Tokens carry aud:"customer" to prevent use against owner endpoints. RFC-0034.',
  overview:
    'The customer-auth module authenticates end-customers (B2B2C users), not workspace owners. It deliberately mirrors the auth module\'s session model but issues JWT tokens with aud:"customer" — this claim prevents customer tokens from being accepted by any middleware that validates owner tokens, and vice versa. Password reset uses a time-limited OTP token delivered via customer_notifications. The CustomerRequired middleware validates customer JWTs for handler protection.',
  install: `import "github.com/agile-factory/kernel/platform/customer_auth"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, customer_auth.Migrations)
}`,
  config: [
    { name: 'DB', type: '*pgxpool.Pool', required: false, description: 'Database pool for transactional writes. May be nil in unit tests.' },
    { name: 'CredRepo', type: 'domain.CredentialRepository', required: true, description: 'Persistence port for customer credentials (hashed password).' },
    { name: 'RefreshRepo', type: 'domain.CustomerRefreshTokenRepository', required: true, description: 'Rotating refresh token store for customer sessions.' },
    { name: 'ResetRepo', type: 'domain.CustomerResetTokenRepository', required: true, description: 'Time-limited password reset token store.' },
    { name: 'Hasher', type: 'usecase.PasswordHasher', required: true, description: 'Password hashing adapter. Use cainfra.NewBcryptHasher().' },
    { name: 'JWTSvc', type: 'usecase.CustomerJWTService', required: true, description: 'JWT service that signs tokens with aud:"customer". Use cainfra.NewCustomerJWTService(secret, ttl).' },
    { name: 'Publisher', type: 'events.Publisher', required: true, description: 'Outbox publisher for customer_auth.registered and customer_auth.password_reset_requested.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
    { name: 'RefreshTTL', type: 'time.Duration', required: false, description: 'Customer refresh token lifetime. Defaults to 30 days.' },
  ],
  usage: `caMod, err := customer_auth.New(customer_auth.Config{
\tDB:          pool,
\tCredRepo:    cainfra.NewPostgresCredentialRepository(pool),
\tRefreshRepo: cainfra.NewPostgresRefreshTokenRepository(pool),
\tResetRepo:   cainfra.NewPostgresResetTokenRepository(pool),
\tHasher:      cainfra.NewBcryptHasher(),
\tJWTSvc:      cainfra.NewCustomerJWTService(os.Getenv("CUSTOMER_JWT_SECRET"), time.Hour),
\tPublisher:   bus.Publisher(),
\tLogger:      log,
})
caMod.RegisterRoutes(mux)

// Protect customer-only handlers with the CustomerRequired middleware.
mux.Handle("/api/v1/tenants/{id}/portal/", caMod.CustomerRequired(portalHandler))`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires six use cases: Register, Login, Refresh, Logout, ForgotPassword, ResetPassword.' },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts all six customer auth endpoints under /api/v1/tenants/{id}/customers/auth/.' },
    {
      name: 'CustomerRequired',
      signature: 'func (m *Module) CustomerRequired(next http.Handler) http.Handler',
      description: 'HTTP middleware that validates customer JWTs (aud:"customer"). Wrap any handler accessible only to authenticated end-customers.',
      example: `mux.Handle("/api/v1/portal/", caMod.CustomerRequired(http.HandlerFunc(portalHandler)))`,
    },
  ],
  events: [
    {
      name: 'customer_auth.registered',
      direction: 'Published',
      description: 'Emitted when an end-customer registers. Consumed by customer_notifications for the welcome message.',
      payload: `{ "end_customer_id", "tenant_id", "email" }`,
    },
    {
      name: 'customer_auth.password_reset_requested',
      direction: 'Published',
      description: 'Emitted when a customer requests a password reset. Consumed by customer_notifications to send the OTP code.',
      payload: `{ "end_customer_id", "tenant_id", "email", "reset_token": "otp_01HF..." }`,
    },
  ],
  endpoints: [
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/register',
      auth: 'none',
      body: `{ "email": "customer@user.io", "password": "s3cr3t" }`,
      responses: [
        { status: '201', body: `{ "customer_id", "email" }` },
        { status: '409', body: `{ "error": "email_taken" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/login',
      auth: 'none',
      body: `{ "email": "customer@user.io", "password": "s3cr3t" }`,
      responses: [
        { status: '200', body: `{ "access_token", "refresh_token", "aud": "customer" }` },
        { status: '401', body: `{ "error": "invalid_credentials" }` },
        { status: '403', body: `{ "error": "customer_suspended" }` },
      ],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/refresh',
      auth: 'none',
      body: `{ "refresh_token": "rftok_01HF..." }`,
      responses: [{ status: '200', body: `{ "access_token", "refresh_token" }` }],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/logout',
      auth: 'Bearer (customer)',
      body: `{ "refresh_token": "rftok_01HF..." }`,
      responses: [{ status: '204', body: `(no content)` }],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/forgot-password',
      auth: 'none',
      body: `{ "email": "customer@user.io" }`,
      responses: [{ status: '204', body: `(OTP sent — always 204 to prevent email enumeration)` }],
    },
    {
      method: 'POST',
      path: '/api/v1/tenants/{id}/customers/auth/reset-password',
      auth: 'none',
      body: `{ "token": "otp_01HF...", "new_password": "newS3cr3t" }`,
      responses: [
        { status: '204', body: `(no content)` },
        { status: '400', body: `{ "error": "invalid_or_expired_token" }` },
      ],
    },
  ],
  migrations: [
    { file: '0001_create_customer_credentials.up.sql', description: 'customer_credentials table with hashed password, reset token, and refresh token store.' },
    { file: '0002_tenant_settings_customer_auth.up.sql', description: 'Extends tenant_settings with customer_auth configuration (token TTL, OTP expiry).' },
  ],
}

// ─── customer-notifications ───────────────────────────────────────────────────

const customerNotifications: ModuleDoc = {
  slug: 'customer-notifications',
  name: 'customer_notifications',
  title: 'customer-notifications',
  description: 'End-customer email + SMS dispatch with per-tenant template overrides and branded content. RFC-0036.',
  overview:
    'The customer-notifications module dispatches transactional emails and SMS to end-customers. It mirrors the notifications module but targets the B2B2C layer: templates can be overridden per-tenant and per-channel, branded content is pulled from white_label, and SMS is supported alongside email. Five outbox events are handled: welcome (register/create), OTP password reset, invoice paid, and payment reminder. Templates are parsed from embed.FS at boot (Lucas amendment). Event handlers are registered on the instance, not global vars (Aziz amendment).',
  install: `import "github.com/agile-factory/kernel/platform/customer_notifications"

func migrate(pool *pgxpool.Pool) error {
\treturn migrator.Run(pool, customer_notifications.Migrations)
}`,
  config: [
    { name: 'DB', type: '*pgxpool.Pool', required: false, description: 'Database pool. May be nil in unit tests.' },
    { name: 'LogRepo', type: 'domain.NotificationLogRepository', required: true, description: 'Persistence port for customer_notification_logs.' },
    { name: 'TemplateRepo', type: 'domain.CustomerTemplateRepository', required: true, description: 'Persistence port for per-tenant template overrides.' },
    { name: 'CustomerResolver', type: 'domain.EndCustomerResolver', required: true, description: 'Resolves end-customer email/phone from their ID — used to look up delivery address.' },
    { name: 'BrandResolver', type: 'domain.BrandResolver', required: true, description: 'Resolves per-tenant brand data (logo, colors) to inject into templates.' },
    { name: 'EmailSender', type: 'domain.EmailSender', required: true, description: 'Email delivery adapter.' },
    { name: 'SMSSender', type: 'domain.SMSSender', required: true, description: 'SMS delivery adapter. Use cninfra.NewNoopSMSAdapter(log) if SMS is not needed.' },
    { name: 'Checker', type: 'permission.Checker', required: true, description: 'Permission checker for admin template management endpoints.' },
    { name: 'Logger', type: '*slog.Logger', required: true, description: 'Structured logger.' },
  ],
  usage: `cnMod, err := customer_notifications.New(customer_notifications.Config{
\tDB:               pool,
\tLogRepo:          cninfra.NewPostgresNotificationLogRepository(pool),
\tTemplateRepo:     cninfra.NewPostgresCustomerTemplateRepository(pool),
\tCustomerResolver: cninfra.NewPostgresEndCustomerResolver(pool),
\tBrandResolver:    cninfra.NewPostgresBrandResolver(pool),
\tEmailSender:      mySendGridAdapter,
\tSMSSender:        cninfra.NewNoopSMSAdapter(log),
\tChecker:          permission.New(memberRepo),
\tLogger:           log,
})

// Register all five event handlers before starting the bus.
cnMod.RegisterHandlers(bus)

cnMod.RegisterRoutes(mux)`,
  methods: [
    { name: 'New', signature: 'func New(cfg Config) (*Module, error)', description: 'Wires the module. Parses all embedded templates at boot so syntax errors surface at startup, not at runtime.' },
    {
      name: 'RegisterHandlers',
      signature: 'func (m *Module) RegisterHandlers(sub events.Subscriber)',
      description: 'Subscribes all five event handlers to the outbox bus. Call before bus.Run(). Handlers are instance methods — not global vars — per the Aziz amendment.',
    },
    { name: 'RegisterRoutes', signature: 'func (m *Module) RegisterRoutes(mux *http.ServeMux)', description: 'Mounts template management and notification log endpoints.' },
  ],
  events: [
    {
      name: 'end_customer.created',
      direction: 'Consumed',
      description: 'Triggers the welcome message on the email (and SMS if configured) channel.',
      payload: `{ "end_customer_id", "tenant_id" }`,
    },
    {
      name: 'customer_auth.registered',
      direction: 'Consumed',
      description: 'Also triggers the welcome message (self-registered customer path).',
      payload: `{ "end_customer_id", "tenant_id" }`,
    },
    {
      name: 'customer_auth.password_reset_requested',
      direction: 'Consumed',
      description: 'Sends the OTP reset code to the customer.',
      payload: `{ "end_customer_id", "tenant_id", "reset_token" }`,
    },
    {
      name: 'billing.invoice_paid',
      direction: 'Consumed',
      description: 'Sends an invoice receipt to the customer.',
      payload: `{ "end_customer_id", "tenant_id", "amount", "currency" }`,
    },
    {
      name: 'billing.payment_reminder',
      direction: 'Consumed',
      description: 'Sends a payment reminder to the customer.',
      payload: `{ "end_customer_id", "tenant_id", "amount", "currency" }`,
    },
  ],
  endpoints: [
    {
      method: 'GET',
      path: '/api/v1/tenants/{id}/customer-notification-logs',
      auth: 'Bearer (ADMIN)',
      responses: [{ status: '200', body: `{ "items": [{ "id", "event_type", "channel", "status", "sent_at" }], "next_cursor" }` }],
    },
    {
      method: 'PUT',
      path: '/api/v1/tenants/{id}/customer-templates/{key}/{channel}',
      auth: 'Bearer (OWNER/ADMIN)',
      body: `{ "subject": "Welcome to {{.TenantName}}", "html": "<p>Hi {{.Name}}</p>", "text": "Hi {{.Name}}" }`,
      responses: [{ status: '200', body: `{ "key", "channel", "updated_at" }` }],
    },
    {
      method: 'DELETE',
      path: '/api/v1/tenants/{id}/customer-templates/{key}/{channel}',
      auth: 'Bearer (OWNER/ADMIN)',
      responses: [{ status: '204', body: `(no content — reverts to default embedded template)` }],
    },
  ],
  migrations: [
    { file: '0001_create_customer_notification_logs.up.sql', description: 'customer_notification_logs table with status, channel, and delivery tracking.' },
    { file: '0002_create_customer_templates.up.sql', description: 'customer_templates table for per-tenant, per-channel template overrides.' },
    { file: '0003_tenant_settings_sms_provider.up.sql', description: 'Extends tenant_settings with SMS provider configuration (provider, API key ref).' },
  ],
}

// ─── exports ──────────────────────────────────────────────────────────────────

export const modules: Record<string, ModuleDoc> = Object.fromEntries(
  [audit, auth, tenant, membership, billing, notifications, user, storage, endCustomers, whiteLabel, customerAuth, customerNotifications].map((m) => [m.slug, m]),
)

export function getModule(slug: string): ModuleDoc | undefined {
  return modules[slug]
}
