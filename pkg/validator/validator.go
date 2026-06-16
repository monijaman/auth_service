package validator

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Struct(s interface{}) error {
	return validate.Struct(s)
}

func FirstError(err error) string {
	if err == nil {
		return ""
	}
	var errs validator.ValidationErrors
	if ok := errors.As(err, &errs); ok && len(errs) > 0 {
		fe := errs[0]
		return fe.Field() + ": " + fe.Tag()
	}
	return err.Error()
}
