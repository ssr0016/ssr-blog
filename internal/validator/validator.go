package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	passwordvalidator "github.com/wagslane/go-password-validator"
)

type CustomValidator struct {
	validator *validator.Validate
}

func New() *CustomValidator {
	v := &CustomValidator{validator: validator.New()}
	_ = v.validator.RegisterValidation("strongpass", validateStrongPassword)
	_ = v.validator.RegisterValidation("notblank", validateNotBlank)
	return v
}

// validateNotBlank rejects strings that are empty or contain only whitespace.
// It does not modify the value, so stored content stays exactly as written.
func validateNotBlank(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	err := passwordvalidator.Validate(password, 60)
	return err == nil
}

var _ echo.Validator = (*CustomValidator)(nil)
