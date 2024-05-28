package eventrouter

import "context"

type contextKey struct{}

var subjectCtxKey = contextKey{}

// SaveSubjectToContext saves the subject to the context
func SaveSubjectToContext(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, subjectCtxKey, subject)
}

// GetSubjectFromContext gets the subject from the context
func GetSubjectFromContext(ctx context.Context) string {
	subject, ok := ctx.Value(subjectCtxKey).(string)
	if !ok {
		return ""
	}

	return subject
}
