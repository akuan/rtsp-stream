package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"

	"rtsp-stream/core/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuthWithSecret(t *testing.T) {
	spec := config.InitConfig()
	provider, err := NewJWTProvider(spec.Auth)
	assert.Nil(t, err)
	//token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"secret":spec.Auth.JWTSecret})
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"secret":"macilaci"})
	fmt.Printf("JWTSecret is %s \n",spec.Auth.JWTSecret)
	tokenString, err := token.SignedString([]byte(spec.Auth.JWTSecret))
	//tokenString, err := token.SignedString([]byte("bhspace"))
	fmt.Printf("tokenString is [%s] \n",tokenString)
	assert.Nil(t, err)
	validated, c := provider.Validate(tokenString)
	assert.NotNil(t, validated)
	fmt.Printf(" The claim.Secret is %s \n",c.Secret)
	validated, c = provider.Validate(fmt.Sprintf("Bearer %s", tokenString))
	assert.NotNil(t, validated)
	fmt.Printf(" The claim.Secret is %s \n",c.Secret)
}

func TestJWTAuthWithRSA(t *testing.T) {
	reader := rand.Reader
	bitSize := 2048
	key, err := rsa.GenerateKey(reader, bitSize)
	assert.Nil(t, err)
	publicKey := key.PublicKey
	provider := JWTProvider{verifyKey: &publicKey}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{})
	tokenString, err := token.SignedString(key)
	fmt.Printf("TestJWTAuthWithRSA tokenString is [%s]",tokenString)
	assert.Nil(t, err)
	validated, _ := provider.Validate(tokenString)
	assert.NotNil(t, validated)
}
