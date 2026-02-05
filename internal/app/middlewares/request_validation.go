package middlewares

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
)

// OapiRequestValidator создает middleware для валидации запросов по OpenAPI.
func OapiRequestValidator(specPath string) (gin.HandlerFunc, error) {
	const methodCtx = "middlewares.OapiRequestValidator"

	if specPath == "" {
		return nil, fmt.Errorf("%s: путь к спецификации не задан", methodCtx)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка чтения спецификации: %w", methodCtx, err)
	}

	swagger, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка разбора спецификации: %w", methodCtx, err)
	}

	// Отключаем проверку host/servers, чтобы валидатор не ломал локальные окружения.
	swagger.Servers = nil

	validator := ginmiddleware.OapiRequestValidatorWithOptions(swagger, &ginmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		SilenceServersWarning: true,
	})

	return validator, nil
}
