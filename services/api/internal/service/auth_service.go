package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"github.com/yourusername/immobilier-saas/services/api/internal/domain"
	"github.com/yourusername/immobilier-saas/services/api/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService struct {
	userRepo       *repository.UserRepository
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	jwtExpiry      time.Duration
	refreshExpiry  time.Duration
}

type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(
	userRepo *repository.UserRepository,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	jwtExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		privateKey:    privateKey,
		publicKey:     publicKey,
		jwtExpiry:     jwtExpiry,
		refreshExpiry: 7 * 24 * time.Hour, // 7 days
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, repository.ErrUserAlreadyExists
	}
	if err != nil && err != repository.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	passwordHash, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate verification token
	verificationToken := s.generateSecureToken()

	// Create user
	user := &domain.User{
		Email:             req.Email,
		PasswordHash:      passwordHash,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Phone:             req.Phone,
		Role:              req.Role,
		VerificationToken: &verificationToken,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: Send verification email (implement email service)
	// s.emailService.SendVerificationEmail(user.Email, verificationToken)

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Check user status
	if user.Status == domain.StatusSuspended {
		return nil, ErrAccountSuspended
	}

	// For MVP: Allow login without email verification
	// In production, you may want to enforce this:
	// if !user.EmailVerified {
	// 	return nil, ErrEmailNotVerified
	// }

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail the login
		fmt.Printf("failed to update last login: %v\n", err)
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

// RefreshToken generates a new access token from a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*domain.LoginResponse, error) {
	// Hash the refresh token to look it up
	tokenHash := s.hashToken(refreshTokenString)

	// Get refresh token from database
	refreshToken, err := s.userRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if err == repository.ErrTokenNotFound {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check user status
	if user.Status == domain.StatusSuspended {
		return nil, ErrAccountSuspended
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString, // Return the same refresh token
		User:         *user,
	}, nil
}

// Logout revokes a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	tokenHash := s.hashToken(refreshTokenString)
	return s.userRepo.RevokeRefreshToken(ctx, tokenHash)
}

// ForgotPassword initiates password reset
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	// Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == repository.ErrUserNotFound {
			// Don't reveal if email exists
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Generate reset token
	resetToken := s.generateSecureToken()
	expiresAt := time.Now().Add(1 * time.Hour) // 1 hour expiry

	// Save reset token
	if err := s.userRepo.SetResetPasswordToken(ctx, user.Email, resetToken, expiresAt); err != nil {
		return fmt.Errorf("failed to set reset token: %w", err)
	}

	// TODO: Send password reset email
	// s.emailService.SendPasswordResetEmail(user.Email, resetToken)

	return nil
}

// ResetPassword resets user password with token
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Get user by reset token
	user, err := s.userRepo.GetByResetToken(ctx, token)
	if err != nil {
		if err == repository.ErrTokenNotFound {
			return ErrInvalidToken
		}
		return fmt.Errorf("failed to get user by reset token: %w", err)
	}

	// Hash new password
	passwordHash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, user.ID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all refresh tokens for security
	if err := s.userRepo.RevokeAllUserTokens(ctx, user.ID); err != nil {
		// Log error but don't fail
		fmt.Printf("failed to revoke all tokens: %v\n", err)
	}

	return nil
}

// VerifyEmail verifies user email with token
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	// Get user by verification token
	user, err := s.userRepo.GetByVerificationToken(ctx, token)
	if err != nil {
		if err == repository.ErrTokenNotFound {
			return ErrInvalidToken
		}
		return fmt.Errorf("failed to get user by verification token: %w", err)
	}

	// Update email verified
	if err := s.userRepo.UpdateEmailVerified(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to update email verified: %w", err)
	}

	return nil
}

// ValidateToken validates a JWT access token
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// Private helper methods

func (s *AuthService) generateAccessToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "immobilier-saas",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID uuid.UUID, deviceInfo, ipAddress string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	tokenString := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token for storage
	tokenHash := s.hashToken(tokenString)

	// Store in database
	refreshToken := &domain.RefreshToken{
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceInfo: &deviceInfo,
		IPAddress:  &ipAddress,
		ExpiresAt:  time.Now().Add(s.refreshExpiry),
	}

	if err := s.userRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) hashPassword(password string) (string, error) {
	// Argon2id parameters (OWASP recommendations)
	const (
		time    = 1
		memory  = 64 * 1024 // 64 MB
		threads = 4
		keyLen  = 32
	)

	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	// Encode as base64 with salt
	result := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		time,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return result, nil
}

func (s *AuthService) verifyPassword(password, hashedPassword string) bool {
	// Parse hash format
	var version, memory, time, threads uint32
	var salt, hash string

	_, err := fmt.Sscanf(
		hashedPassword,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		&version,
		&memory,
		&time,
		&threads,
		&salt,
		&hash,
	)
	if err != nil {
		return false
	}

	// Decode salt and hash
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}

	// Hash the input password with same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		saltBytes,
		time,
		memory,
		uint8(threads),
		uint32(len(hashBytes)),
	)

	// Constant-time comparison
	if len(hashBytes) != len(computedHash) {
		return false
	}

	var diff byte
	for i := range hashBytes {
		diff |= hashBytes[i] ^ computedHash[i]
	}
	return diff == 0
}

func (s *AuthService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *AuthService) generateSecureToken() string {
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	return base64.URLEncoding.EncodeToString(tokenBytes)
}
