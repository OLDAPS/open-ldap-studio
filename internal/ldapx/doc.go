// Package ldapx is the only package in this application that speaks LDAP.
//
// Boundary: no other package may dial, bind, search, or modify. Its mutation functions (Add, Modify, ModRDN, Delete) have exactly one calling package, internal/changeset — asserted by contract test C1.
package ldapx
