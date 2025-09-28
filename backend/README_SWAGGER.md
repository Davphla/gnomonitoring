# Swagger API Documentation

## 🚀 Quick Start

The API now uses **Swagger/OpenAPI** for comprehensive, interactive documentation!

## 📖 Access Documentation

1. **Start the server:**
   ```bash
   go run main.go
   ```

2. **Open Swagger UI:**
   - Visit: `http://localhost:8080/swagger/index.html`
   - Root URL `http://localhost:8080/` automatically redirects to Swagger

## 🛠️ Development Mode

When `dev_mode: true` in your config:
- Use `X-Debug-UserID` header for authentication
- No need for Clerk tokens during development

## 📝 Adding New Endpoints

To document new endpoints, add Swagger annotations above your handler functions:

```go
// CreateWebhook godoc
// @Summary Create governance webhook
// @Description Create a new governance webhook for the authenticated user
// @Tags governance-webhooks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security DevAuth
// @Param webhook body database.WebhookGovDAO true "Webhook data"
// @Success 201 {object} map[string]string "Webhook created successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Router /webhooks/govdao [post]
func CreateWebhookHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
    // Implementation...
}
```

## 🔄 Regenerate Documentation

After adding/modifying annotations:

```bash
swag init
```

This updates the generated files in the `docs/` directory.

## 📊 Features

- **Interactive Testing**: Test API endpoints directly from the browser
- **Authentication Support**: Both production (Bearer tokens) and development modes
- **Request/Response Examples**: See exactly what data structures are expected
- **Automatic Validation**: Parameter and response validation
- **Export Options**: Download OpenAPI spec as JSON/YAML

## 🎯 Benefits over Custom HTML

- ✅ **Interactive**: Test endpoints directly in the browser
- ✅ **Standardized**: OpenAPI 3.0 specification
- ✅ **Auto-generated**: Always in sync with your code
- ✅ **Professional**: Industry-standard documentation
- ✅ **Client Generation**: Generate SDKs in multiple languages
- ✅ **Validation**: Built-in request/response validation
