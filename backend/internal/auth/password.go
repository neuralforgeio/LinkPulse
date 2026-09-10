// Package auth implements authentication: password hashing and the
// register flow. Login/refresh/logout arrive in the next session.
package auth

import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "fmt"
    "strings"

    "golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP-recommended minimums). m = memory in KiB,
// t = iterations, p = parallelism.
const (
    argon2Time    = 2
    argon2Memory  = 19456 // 19 MiB
    argon2Threads = 1
    argon2KeyLen  = 32
    argon2SaltLen = 16
)

// HashPassword derives an Argon2id hash of the password, returned in the
// PHC string format: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>.
// Parameters are embedded in the string so VerifyPassword can reproduce
// them exactly — meaning parameters can change later without breaking
// existing stored hashes.
func HashPassword(password string) (string, error) {
    salt := make([]byte, argon2SaltLen)
    if _, err := rand.Read(salt); err != nil {
        return "", fmt.Errorf("generate salt: %w", err)
    }

    hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

    return fmt.Sprintf(
        "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version, argon2Memory, argon2Time, argon2Threads,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    ), nil
}

// VerifyPassword reports whether the password matches a stored PHC-format
// Argon2id hash. Wrong password returns (false, nil) — that is a normal
// outcome, not an error.
func VerifyPassword(password, encoded string) (bool, error) {
    parts := strings.Split(encoded, "$")
    if len(parts) != 6 || parts[1] != "argon2id" {
        return false, fmt.Errorf("unrecognized hash format")
    }

    var version int
    if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
        return false, fmt.Errorf("parse hash version: %w", err)
    }
    if version != argon2.Version {
        return false, fmt.Errorf("unsupported argon2 version %d", version)
    }

    var memory, time uint32
    var threads uint8
    if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
        return false, fmt.Errorf("parse hash params: %w", err)
    }

    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return false, fmt.Errorf("decode salt: %w", err)
    }

    want, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return false, fmt.Errorf("decode hash: %w", err)
    }

    got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))

    // Constant-time compare prevents timing attacks.
    return subtle.ConstantTimeCompare(got, want) == 1, nil
}
