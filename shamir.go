package main

import (
	"crypto/rand"
	"log"

	"github.com/etiennebch/shamir-sss/galois"
	"github.com/etiennebch/shamir-sss/random"
)

// Split splits a secret of length p into n shares using Shamir secret sharing scheme, such
// that at least 2 <= k <= n shares (known as the threshold) must be combined in order to recover
// the secret.
// We refer to such a scheme as a (k,n) Shamir scheme.
//
// All computation is done in the Galois finite field 2^8 - GF(2^8) - as it is convenient for
// byte-oriented computation, and is the de-facto field used by the AES cipher.
// The maximum number of shares that can be dealt is the 2^8-1.
//
// The secret is processed one byte at a time. Every byte of the secret is split using Shamir's scheme.
// In a (k,n) Shamir scheme, each byte of the secret yields n "mini-shares".
// Thus, every participant in the scheme receives a share, which is a collection of the p mini-shares
// attributed to him and an additional value (see below).
//
// The result of Split is a share matrix of dimensions [(p+1) * n]. Each column of the matrix is a
// participant's secret. For every column i, the first p components make the share of participant i.
// Each component encodes is the share of the corresponding byte of the secret.
// The last component is the coordinate x[i] used to evaluate the polynomials for participant i.
//
// Using the same point across mini-shares does not reduce security so long as we still use distinct
// points for distinct participants.
//
// Note that Shamir secret sharing scheme is secure in that with less than k shares, no adversary can
// learn anything about the secret. However, Shamir's scheme does leak the size of the secret
// since the length of the share is p + 1, unless the secret is padded somehow.
// Padding the secret would still leak the information that the secret is at most the length of the
// padded secret + 1.
//
// For large secrets, a common approach is to first encrypt the secret using a strong cipher, and to
// use Shamir secret sharing on the decryption key rather than on the underlying secret.
//
// The algorithm used is as follows:
//
// For every byte chunk c of the secret of length p, a random polynomial with coefficients in GF(2^8) is picked.
// 	- The polynomial's intercept is set to c.
// 	- Then, we pick n distinct points from GF(2^8) such that each participant is assigned a unique
// 	  point x[i], 0 <= i <= n <= 255.
// 	- Then, we evaluate the polynomial for all x[i] and the resulting value y is the share of c for
//	  participant i. y[i] is added to the result share matrix.
//
// For all participants i, append x[i] to the corresponding column in the share matrix.
// Recipient i would receive the column [y[0], y[1], ... y[p-1], x[i]].
// Return the share matrix.
func Split(secret []byte, n, threshold uint8) ([][]byte, error) {
	if threshold > n {
		log.Fatal("the threshold value cannot be greater than the number of shares to deal.")
	}
	if len(secret) == 0 {
		log.Fatal("the secret cannot be empty.")
	}
	if threshold < 2 {
		log.Fatal("the threshold value must be at least 2.")
	}

	shares := initShareMatrix(n, uint(len(secret)))
	x := pickCoordinates(n)

	for idx, chunk := range secret {
		polynomial, err := randomPolynomial(threshold)
		if err != nil {
			// TODO: timing side-channel attack possible ?
			// error message not included in the log to avoid leaking sensitive information.
			log.Fatalf("failed to generate random polynomial.")
		}
		// set the polynomial intercept to the secret chunk
		polynomial[0] = chunk
		// compute the value of the polynomial for every coordinate x[i]
		for i := 0; uint8(i) < n; i++ {
			share := evaluatePolynomial(x[i], polynomial)
			shares[i][idx] = share
		}
	}

	// append the point x[i] to each participant's share.
	for i := 0; uint8(i) < n; i++ {
		shares[i][len(secret)] = x[i]
	}
	return shares, nil
}

// func Recover() {}

// randomPolynomial generates a polynomial of the provided order with random coefficients in GF(2^8)
// In the context of a (k,n) Shamir scheme, the polynomial order must be k. As we use GF(2^8),
// the maximum polynomial order is the maximum number of distributable shares, that is 2^8-1.
func randomPolynomial(order uint8) ([]byte, error) {
	coefficients := make([]byte, order)
	_, err := rand.Read(coefficients[1:])
	if err != nil {
		return nil, err
	}
	return coefficients, nil
}

// pickCoordinates picks n distinct point in GF(2^8).
// As we operate in GF(2^8), it holds that 0 <= n <= 255.
func pickCoordinates(n uint8) []byte {
	coordinates := make([]byte, n, n)
	permutation := random.PermSecure(int(n))
	for i, x := range permutation {
		coordinates[i] = byte(x)
	}
	return coordinates
}

// evaluatePolynomial computes the value of a polynomial at point x, using Horner's algorithm.
func evaluatePolynomial(x byte, polynomial []byte) byte {
	if x == 0 {
		return polynomial[0]
	}

	degree := len(polynomial) - 1
	// initialize Horner's algorithm with the nth coefficient of the polynomial
	value := polynomial[degree]
	field := galois.NewField256()
	for i := degree - 1; i >= 0; i-- {
		value = field.Add(polynomial[i], field.Multiply(value, x))
	}
	return value
}

// initShareMatrix initializes an empty share matrix.
// the matrix is of dimensions [(secretLength+1) * n]
func initShareMatrix(n uint8, secretLength uint) [][]byte {
	matrix := make([][]byte, n, n)
	for i := range matrix {
		matrix[i] = make([]byte, secretLength+1, secretLength+1)
	}
	return matrix
}
