package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "password123"
	hash, err := hashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPassword(t *testing.T) {
	password := "password123"
	hash, err := hashPassword(password)
	assert.NoError(t, err)

	// Test correct password
	err = checkPassword(hash, password)
	assert.NoError(t, err)

	// Test incorrect password
	err = checkPassword(hash, "wrongpassword")
	assert.Error(t, err)
}
