package register

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/internal/service/event"
	"github.com/monir/auth_service/internal/service/password"
)

var (
	ErrEmailTaken = errors.New("email already registered")
)

type Input struct {
	Email    string `json:"email"    validate:"required,email"`
	Phone    string `json:"phone"    validate:"omitempty,e164"`
	Password string `json:"password" validate:"required,min=8"`
}

type Output struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

type UseCase struct {
	userRepo user.Repository
	pwdSvc   *password.Service
	eventPub event.Publisher
}

func New(userRepo user.Repository, pwdSvc *password.Service, eventPub event.Publisher) *UseCase {
	return &UseCase{userRepo: userRepo, pwdSvc: pwdSvc, eventPub: eventPub}
}

func (uc *UseCase) Execute(ctx context.Context, in Input) (*Output, error) {
	hash, err := uc.pwdSvc.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	u := &user.User{
		ID:            uuid.New(),
		Email:         in.Email,
		Phone:         in.Phone,
		PasswordHash:  hash,
		EmailVerified: false,
		Status:        user.StatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := uc.userRepo.Create(ctx, u); err != nil {
		if errors.Is(err, postgres.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	_ = uc.eventPub.Publish(ctx, event.Event{
		Type:      event.UserRegistered,
		Payload:   map[string]string{"user_id": u.ID.String(), "email": u.Email},
		OccuredAt: now,
	})

	return &Output{UserID: u.ID, Email: u.Email}, nil
}
