// utils/password - Provides bcrypt password hashing and comparison.
// Passwords are never stored in plain text; only their hashes are saved.
package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the given plain-text password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword returns true if the plain-text password matches the hash.
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
