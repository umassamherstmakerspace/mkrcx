package leash_authentication

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func InitalizeJWT() Keys {
	//read text from file keyfile
	key_file := os.Getenv("KEY_FILE")
	if _, err := os.Stat(key_file); os.IsNotExist(err) {
		raw, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatal(err)
		}

		key, err := jwk.FromRaw(raw)
		if err != nil {
			log.Fatal(err)
		}

		key.Set(jwk.KeyIDKey, "sig-"+strconv.FormatInt(time.Now().Unix(), 10))
		key.Set(jwk.AlgorithmKey, jwa.RS256)
		key.Set(jwk.KeyUsageKey, jwk.ForSignature)

		keys := jwk.NewSet()
		keys.AddKey(key)

		buf, err := json.MarshalIndent(keys, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(key_file, buf, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	keyFile, err := os.Open(key_file)
	if err != nil {
		log.Fatal(err)
	}

	defer keyFile.Close()

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	keys, err := jwk.Parse(keyBytes)
	if err != nil {
		fmt.Printf("failed to parse private key: %s\n", err)
	}

	privateKey, _ := keys.Key(0)

	publicKey, err := privateKey.PublicKey()
	if err != nil {
		fmt.Printf("failed to get public key: %s\n", err)
	}

	return Keys{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (keys Keys) GetPublicKey() jwk.Key {
	return keys.publicKey
}

func (keys Keys) GetPrivateKey() jwk.Key {
	return keys.privateKey
}

func (keys Keys) Sign(token jwt.Token) ([]byte, error) {
	return jwt.Sign(token, jwt.WithKey(jwa.RS256, keys.privateKey))
}

func (keys Keys) Parse(token string) (jwt.Token, error) {
	tok, err := jwt.ParseString(token, jwt.WithKey(jwa.RS256, keys.publicKey))
	if err != nil {
		return nil, err
	}

	if err := jwt.Validate(tok); err != nil {
		return nil, err
	}

	return tok, nil
}
