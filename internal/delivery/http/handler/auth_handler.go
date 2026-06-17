package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	mw "github.com/monir/auth_service/internal/middleware"
	"github.com/monir/auth_service/internal/usecase/forgotpassword"
	"github.com/monir/auth_service/internal/usecase/login"
	"github.com/monir/auth_service/internal/usecase/logout"
	"github.com/monir/auth_service/internal/usecase/refresh"
	"github.com/monir/auth_service/internal/usecase/register"
	"github.com/monir/auth_service/internal/usecase/resetpassword"
	"github.com/monir/auth_service/internal/usecase/verifyemail"
	"github.com/monir/auth_service/pkg/response"
)

type AuthHandler struct {
	registerUC      *register.UseCase
	loginUC         *login.UseCase
	refreshUC       *refresh.UseCase
	logoutUC        *logout.UseCase
	forgotPwUC      *forgotpassword.UseCase
	resetPwUC       *resetpassword.UseCase
	verifyEmailUC   *verifyemail.UseCase
}

func NewAuthHandler(
	registerUC *register.UseCase,
	loginUC *login.UseCase,
	refreshUC *refresh.UseCase,
	logoutUC *logout.UseCase,
	forgotPwUC *forgotpassword.UseCase,
	resetPwUC *resetpassword.UseCase,
	verifyEmailUC *verifyemail.UseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUC:    registerUC,
		loginUC:       loginUC,
		refreshUC:     refreshUC,
		logoutUC:      logoutUC,
		forgotPwUC:    forgotPwUC,
		resetPwUC:     resetPwUC,
		verifyEmailUC: verifyEmailUC,
	}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var in register.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.registerUC.Execute(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, register.ErrEmailTaken) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, register.ErrSiteRequired) || errors.Is(err, register.ErrSiteNotFound) || errors.Is(err, register.ErrRoleNotFound) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.Created(c, out)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var in login.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.loginUC.Execute(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, login.ErrInvalidCredentials) {
			response.Unauthorized(c, err.Error())
			return
		}
		if errors.Is(err, login.ErrAccountInactive) {
			c.JSON(http.StatusForbidden, response.Body{Success: false, Error: err.Error()})
			return
		}
		if errors.Is(err, login.ErrSiteRequired) || errors.Is(err, login.ErrSiteNotFound) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, login.ErrNoSiteAccess) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, out)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var in refresh.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.refreshUC.Execute(c.Request.Context(), in)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.OK(c, out)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var in logout.Input
	_ = c.ShouldBindJSON(&in)

	claims := mw.ClaimsFromContext(c)
	if claims != nil {
		in.UserID = claims.UserID
	}

	if err := h.logoutUC.Execute(c.Request.Context(), in); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var in forgotpassword.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	_ = h.forgotPwUC.Execute(c.Request.Context(), in)
	// Always return 200 to prevent email enumeration
	response.OK(c, gin.H{"message": "if the email exists, a reset code has been sent"})
}

// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var in resetpassword.Input
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.resetPwUC.Execute(c.Request.Context(), in); err != nil {
		if errors.Is(err, resetpassword.ErrInvalidOTP) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"message": "password reset successful"})
}

// POST /api/v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var in verifyemail.VerifyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	claims := mw.ClaimsFromContext(c)
	if claims != nil {
		in.UserID = claims.UserID
	}
	if err := h.verifyEmailUC.Verify(c.Request.Context(), in); err != nil {
		if errors.Is(err, verifyemail.ErrInvalidOTP) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"message": "email verified"})
}

// POST /api/v1/auth/resend-verification
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	claims := mw.ClaimsFromContext(c)
	if claims == nil {
		response.Unauthorized(c, "not authenticated")
		return
	}
	if err := h.verifyEmailUC.SendCode(c.Request.Context(), verifyemail.SendInput{UserID: claims.UserID}); err != nil {
		if errors.Is(err, verifyemail.ErrAlreadyVerified) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"message": "verification code sent"})
}
