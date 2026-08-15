package handlers

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

// https://regex101.com/r/Ly7O1x/3/
var semverRegex = regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func init() {
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
	Validate.RegisterValidation("semver", validateSemver)
	Validate.RegisterValidation("rollout_stages", validateRolloutStages)
	Validate.RegisterValidation("update_attempt_result", validateUpdateAttemptResult)
}

func validateSemver(fl validator.FieldLevel) bool {
	return semverRegex.MatchString(fl.Field().String())
}

func validateRolloutStages(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.Slice {
		return false
	}

	indexes := make([]int, field.Len())
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)
		oi := elem.FieldByName("OrderIndex")
		if !oi.IsValid() || oi.Kind() != reflect.Int {
			return false
		}
		indexes[i] = int(oi.Int())
	}

	slices.Sort(indexes)

	for i, idx := range indexes {
		if idx != i {
			return false
		}
	}
	return true
}

func validateUpdateAttemptResult(fl validator.FieldLevel) bool {
	resultString := fl.Field().String()
	result := domain.UpdateAttemptsResult(resultString)
	return result.IsValid()
}

func FormatValidation(valErrs validator.ValidationErrors) map[string]string {
	messages := make(map[string]string, len(valErrs))

	for _, fe := range valErrs {
		field := fe.Field()

		switch fe.Tag() {
		case "required":
			messages[field] = "Field is required"
		case "min":
			messages[field] = fmt.Sprintf("Field must be at least %s", fe.Param())
		case "max":
			messages[field] = fmt.Sprintf("Field must be at most %s", fe.Param())
		case "gt":
			messages[field] = fmt.Sprintf("Field must be greater than %s", fe.Param())
		case "len":
			messages[field] = fmt.Sprintf("Field must be exactly %s", fe.Param())
		case "url":
			messages[field] = "Field must be a valid URL"
		case "uuid":
			messages[field] = "Field must be a valid UUID"
		case "hexadecimal":
			messages[field] = "Field must be a valid hexadecimal string"
		case "semver":
			messages[field] = "Field must be a valid semver string"
		case "rollout_stages":
			messages[field] = "Stages should form 0..n-1 row"
		case "update_attempt_result":
			results := domain.GetAllValidUpdateAttemptsResults()
			quoted := make([]string, len(results))
			for i, r := range results {
				quoted[i] = fmt.Sprintf("'%s'", r)
			}

			messages[field] = fmt.Sprintf("Result must be one of %s", strings.Join(quoted, ", "))
		default:
			messages[field] = "Field is invalid"
		}
	}

	return messages
}
