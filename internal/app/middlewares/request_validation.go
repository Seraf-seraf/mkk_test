package middlewares

import (
	"fmt"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

// OapiRequestValidator создает middleware для валидации запросов по OpenAPI.
func OapiRequestValidator(specPath string) (gin.HandlerFunc, error) {
	const methodCtx = "middlewares.OapiRequestValidator"

	if specPath == "" {
		return nil, fmt.Errorf("%s: путь к спецификации не задан", methodCtx)
	}

	validator, err := ginmiddleware.OapiValidatorFromYamlFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return validator, nil
}
