// Package credentials models a Credential: a named, assignable reference to a secret held by the platform.
//
// Boundary: it stores metadata and a SecretRef. It never holds, returns, or logs secret material; retrieval is internal/secrets' job alone.
package credentials
