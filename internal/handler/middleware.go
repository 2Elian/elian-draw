package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware adds CORS headers for Electron/localhost origins
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, x-ai-provider, x-ai-api-key, x-ai-model, x-ai-base-url, x-access-code, x-selected-model-id, x-minimal-style, x-aws-access-key-id, x-aws-secret-access-key, x-aws-region, x-aws-session-token, x-vertex-api-key, cookie")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		/*
			遇到 c.Next()：代码会“暂停”当前中间件的执行，跳去执行下一个中间件，等后面所有的逻辑都跑完了，再回到 c.Next() 下面继续执行（如果有的话）。
		*/
		c.Next()
	}
}

// AccessCodeMiddleware validates the access code if configured
func AccessCodeMiddleware(accessCodes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(accessCodes) == 0 {
			c.Next()
			return
		}

		code := c.GetHeader("x-access-code")
		if code == "" {
			code = c.Query("accessCode")
		}

		if code == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing access code. Please configure it in Settings.",
			})
			return
		}

		found := false
		for _, ac := range accessCodes {
			if ac == code {
				found = true
				break
			}
		}

		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing access code. Please configure it in Settings.",
			})
			return
		}

		c.Next()
	}
}

// ErrorRecoveryMiddleware recovers from panics and returns a JSON error
func ErrorRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Sanitize error message - prevent leaking API keys
				message := fmt.Sprintf("%v", r)
				message = sanitizeErrorMessage(message)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": message,
				})
			}
		}()
		c.Next()
	}
}

// StreamErrorHandler wraps handlers to catch errors and return appropriate responses
func StreamErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// If response hasn't started, send JSON error
				if !c.Writer.Written() {
					message := fmt.Sprintf("%v", r)
					message = sanitizeErrorMessage(message)
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": message,
					})
				}
			}
		}()
		c.Next()
	}
}

// DrainBody reads and restores the request body
func DrainBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func sanitizeErrorMessage(msg string) string {
	lower := strings.ToLower(msg)
	keywords := []string{"key", "token", "sig", "signature", "secret", "password", "credential"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return "Authentication failed. Please check your credentials."
		}
	}
	return msg
}
