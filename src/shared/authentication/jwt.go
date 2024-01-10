package leash_authentication

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Keys struct {
	publicKey  jwk.Key
	privateKey jwk.Key
}

// GenerateJWTKeySet generates a new set of JWT keys
func GenerateJWTKeySet() (jwk.Set, error) {
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	key, err := jwk.FromRaw(raw)
	if err != nil {
		return nil, err
	}

	// Set the key ID, algorithm, and usage
	key.Set(jwk.KeyIDKey, "sig-"+strconv.FormatInt(time.Now().Unix(), 10))
	key.Set(jwk.AlgorithmKey, jwa.RS256)
	key.Set(jwk.KeyUsageKey, jwk.ForSignature)

	keys := jwk.NewSet()
	keys.AddKey(key)

	return keys, nil
}

// CreateOrGetKeysFromFile initalizes the JWT key set from a file
func CreateOrGetKeysFromFile(key_file string) (jwk.Set, error) {
	// Generate a key if it doesn't exist
	if _, err := os.Stat(key_file); os.IsNotExist(err) {
		keys, err := GenerateJWTKeySet()
		if err != nil {
			return nil, err
		}

		// Format the key file
		buf, err := json.MarshalIndent(keys, "", "  ")
		if err != nil {
			return nil, err
		}

		// Write the key file
		err = os.WriteFile(key_file, buf, 0600)
		if err != nil {
			return nil, err
		}

		return keys, nil
	} else if err != nil {
		return nil, err
	}

	// Read the key file
	keyFile, err := os.Open(key_file)
	if err != nil {
		return nil, err
	}

	defer keyFile.Close()

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		return nil, err
	}

	// Parse the key file
	keys, err := jwk.Parse(keyBytes)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// CreateKeys initalizes the keys from a JWK set
func CreateKeys(keys jwk.Set) (*Keys, error) {
	// Get the first key
	privateKey, _ := keys.Key(0)

	// Create the public key from the private key
	publicKey, err := privateKey.PublicKey()
	if err != nil {
		return nil, err
	}

	// Return the keys
	return &Keys{
		publicKey:  publicKey,
		privateKey: privateKey,
	}, nil
}

// GetPublicKey returns the public key
func (keys Keys) GetPublicKey() jwk.Key {
	return keys.publicKey
}

// GetPrivateKey returns the private key
func (keys Keys) GetPrivateKey() jwk.Key {
	return keys.privateKey
}

// Sign signs a token
func (keys Keys) Sign(token jwt.Token) ([]byte, error) {
	return jwt.Sign(token, jwt.WithKey(jwa.RS256, keys.privateKey))
}

// Parse parses and validates a token
func (keys Keys) Parse(token string) (jwt.Token, error) {
	// Parse the token
	tok, err := jwt.ParseString(token, jwt.WithKey(jwa.RS256, keys.publicKey))
	if err != nil {
		return nil, err
	}

	// Validate the token
	if err := jwt.Validate(tok); err != nil {
		return nil, err
	}

	return tok, nil
}
