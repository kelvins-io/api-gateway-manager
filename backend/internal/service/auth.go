package service

import (
	"errors"
	"fmt"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/jwtutil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrBadRequest   = errors.New("bad request")
)

type AuthService struct {
	db     *gorm.DB
	jwtMgr *jwtutil.Manager
}

func NewAuthService(db *gorm.DB, jwtMgr *jwtutil.Manager) *AuthService {
	return &AuthService{db: db, jwtMgr: jwtMgr}
}

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

func (s *AuthService) Register(in RegisterInput) (*AuthResult, error) {
	var count int64
	if err := s.db.Model(&model.User{}).Where("username = ?", in.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: username already exists", ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := s.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, err
	}
	role := model.RoleMember
	if total == 0 {
		role = model.RoleSystemAdmin
	}

	user := &model.User{
		Username:     in.Username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	token, err := s.jwtMgr.Generate(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthService) Login(in LoginInput) (*AuthResult, error) {
	var user model.User
	if err := s.db.Where("username = ?", in.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: invalid username or password", ErrUnauthorized)
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, fmt.Errorf("%w: invalid username or password", ErrUnauthorized)
	}
	token, err := s.jwtMgr.Generate(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: &user}, nil
}

func (s *AuthService) GetUser(id uint64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
