package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/kernel/errors"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	auditusecase "github.com/AgileFactory/kernel/platform/audit/usecase"
	authdomain "github.com/AgileFactory/kernel/platform/auth/domain"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
	tenantdomain "github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/google/uuid"
)

// PasswordHasher defines how passwords should be hashed.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// RegisterUseCase handles the user registration flow.
type RegisterUseCase struct {
	repo           authdomain.UserRepository
	hasher         PasswordHasher
	tenantRepo     tenantdomain.TenantRepository
	membershipRepo memberdomain.MembershipRepository
	audit          *auditusecase.LogEventUseCase
}

// NewRegisterUseCase creates a new RegisterUseCase.
func NewRegisterUseCase(
	repo authdomain.UserRepository,
	hasher PasswordHasher,
	tenantRepo tenantdomain.TenantRepository,
	membershipRepo memberdomain.MembershipRepository,
	audit *auditusecase.LogEventUseCase,
) *RegisterUseCase {
	return &RegisterUseCase{
		repo:           repo,
		hasher:         hasher,
		tenantRepo:     tenantRepo,
		membershipRepo: membershipRepo,
		audit:          audit,
	}
}

// Execute performs the registration.
//
// Steps:
//  1. Validate email format
//  2. Verify the tenant exists and is active
//  3. Check the email is not already taken
//  4. Hash the password
//  5. Persist the User
//  6. Create an ACTIVE self-membership with the specified role
//
// role must be one of ROLE_OWNER, ROLE_ADMIN, ROLE_AGENT, ROLE_VIEWER.
// ROLE_OWNER should only be passed when the registering user is the tenant creator.
func (uc *RegisterUseCase) Execute(ctx context.Context, id, tenantID, emailStr, password string, role memberdomain.MemberRole) (*authdomain.User, error) {
	// 1. Validate email
	email, err := authdomain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	// 2. Verify tenant exists and is active
	tenant, err := uc.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !tenant.IsActive {
		return nil, tenantdomain.ErrTenantNotFound
	}

	// 3. Check email uniqueness
	existingUser, _ := uc.repo.FindByEmail(ctx, email)
	if existingUser != nil {
		return nil, authdomain.ErrUserAlreadyExists
	}

	// 4. Hash password
	hashedPassword, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	// 5. Save the user (no tenant_id field)
	user := authdomain.NewUser(id, email, hashedPassword, authdomain.ROLE_VIEWER)
	if err := uc.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	// 6. Create an ACTIVE self-membership with the provided role
	membershipID := uuid.New().String()
	selfMembership := memberdomain.NewSelfMembership(membershipID, user.ID, tenantID, role)
	if err := uc.membershipRepo.Save(ctx, selfMembership); err != nil {
		return nil, errors.WrapInternal(err)
	}

	uc.audit.Execute(ctx, auditusecase.LogEventParams{
		TenantID:   tenant.ID,
		ActorID:    user.ID,
		Action:     auditdomain.ACTION_USER_REGISTERED,
		TargetType: "user",
		TargetID:   user.ID,
		Metadata: map[string]any{
			"email": user.Email,
		},
	})

	return user, nil
}
