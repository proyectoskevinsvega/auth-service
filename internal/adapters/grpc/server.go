package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	pb "github.com/vertercloud/auth-service/internal/adapters/grpc/proto"
	"github.com/vertercloud/auth-service/internal/domain"
	"github.com/vertercloud/auth-service/internal/ports"
	"github.com/vertercloud/auth-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer implements the gRPC AuthService
type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	tokenUC  *usecase.TokenUseCase
	userRepo ports.UserRepository
	logger   zerolog.Logger
}

// NewAuthServer creates a new gRPC auth server
func NewAuthServer(
	tokenUC *usecase.TokenUseCase,
	userRepo ports.UserRepository,
	logger zerolog.Logger,
) *AuthServer {
	return &AuthServer{
		tokenUC:  tokenUC,
		userRepo: userRepo,
		logger:   logger,
	}
}

// ValidateToken validates a JWT token and returns user info
func (s *AuthServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	clientIdentity := GetClientIdentity(ctx)
	s.logger.Info().Str("client_identity", clientIdentity).Msg("gRPC ValidateToken request received")

	if req.Token == "" {
		return &pb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    "INVALID_TOKEN",
			ErrorMessage: "token is required",
		}, nil
	}

	token, err := s.tokenUC.ValidateToken(ctx, req.Token)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to validate token")

		errorCode := "INVALID_TOKEN"
		errorMessage := "invalid token"

		switch err {
		case domain.ErrTokenExpired:
			errorCode = "TOKEN_EXPIRED"
			errorMessage = "token expired"
		case domain.ErrTokenRevoked:
			errorCode = "TOKEN_REVOKED"
			errorMessage = "token revoked"
		}

		return &pb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    errorCode,
			ErrorMessage: errorMessage,
		}, nil
	}

	// Get user details
	user, err := s.userRepo.GetByID(ctx, token.TenantID, token.UserID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", token.UserID).Msg("failed to get user")
		return &pb.ValidateTokenResponse{
			Valid:        false,
			ErrorCode:    "USER_NOT_FOUND",
			ErrorMessage: "user not found",
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:            true,
		UserId:           user.ID,
		Email:            user.Email,
		Username:         user.Username,
		TenantId:         user.TenantID,
		Active:           user.Active,
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		ErrorCode:        "",
		ErrorMessage:     "",
	}, nil
}

// RevokeToken revokes a token
func (s *AuthServer) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	if err := s.tokenUC.RevokeToken(ctx, req.Token); err != nil {
		s.logger.Error().Err(err).Msg("failed to revoke token")
		return &pb.RevokeTokenResponse{
			Success: false,
			Message: "failed to revoke token",
		}, nil
	}

	return &pb.RevokeTokenResponse{
		Success: true,
		Message: "token revoked successfully",
	}, nil
}

// GetUserByID retrieves user information by ID
func (s *AuthServer) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.GetUserByIDResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.userRepo.GetByID(ctx, req.TenantId, req.UserId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error().Err(err).Str("user_id", req.UserId).Msg("failed to get user")
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &pb.GetUserByIDResponse{
		Id:               user.ID,
		Username:         user.Username,
		Email:            user.Email,
		TenantId:         user.TenantID,
		Active:           user.Active,
		EmailVerified:    user.EmailVerified,
		TwoFactorEnabled: user.TwoFactorEnabled,
		CreatedAt:        user.CreatedAt.Format(time.RFC3339),
	}, nil
}
