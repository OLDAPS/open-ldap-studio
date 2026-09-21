// Command open-ldap-studio starts the Open LDAP Studio desktop application.
//
// This package is the composition root: it embeds the built frontend, constructs
// the application services, binds the bridge to Wails, and owns process startup
// and shutdown. Product behavior belongs in internal packages; this package must
// not implement LDAP operations, persistence rules, or UI workflows.
package main
