package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	jwtSvc "github.com/monir/auth_service/internal/service/jwt"
	authpb "github.com/monir/auth_service/proto/gen/auth"
)

type AuthServer struct {
	jwtSvc   *jwtSvc.Service
	userRepo user.Repository
}

func NewAuthServer(jwtSvc *jwtSvc.Service, userRepo user.Repository) *AuthServer {
	return &AuthServer{jwtSvc: jwtSvc, userRepo: userRepo}
}

func (s *AuthServer) ValidateToken(_ context.Context, req *authpb.TokenRequest) (*authpb.TokenResponse, error) {
	claims, err := s.jwtSvc.ValidateAccessToken(req.Token)
	if err != nil {
		return &authpb.TokenResponse{Valid: false}, nil
	}
	return &authpb.TokenResponse{
		Valid:       true,
		UserID:      claims.UserID.String(),
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}

func (s *AuthServer) GetUser(ctx context.Context, req *authpb.UserRequest) (*authpb.UserResponse, error) {
	id, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id")
	}

	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	roles, _ := s.userRepo.GetRoles(ctx, u.ID)

	return &authpb.UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Phone:         u.Phone,
		Status:        string(u.Status),
		EmailVerified: u.EmailVerified,
		Roles:         roles,
	}, nil
}

func (s *AuthServer) HasPermission(ctx context.Context, req *authpb.PermissionRequest) (*authpb.PermissionResponse, error) {
	id, err := uuid.Parse(req.UserID)
	if err != nil {
		return &authpb.PermissionResponse{HasPermission: false}, nil
	}

	perms, err := s.userRepo.GetPermissions(ctx, id)
	if err != nil {
		return &authpb.PermissionResponse{HasPermission: false}, nil
	}

	for _, p := range perms {
		if p == req.Permission {
			return &authpb.PermissionResponse{HasPermission: true}, nil
		}
	}
	return &authpb.PermissionResponse{HasPermission: false}, nil
}
