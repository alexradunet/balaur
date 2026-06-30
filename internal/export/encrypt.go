package export

import (
	"archive/tar"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// The encrypted-export envelope (plan 195): one passphrase-encrypted archive
// over the already-written Markdown mirror tree. It is the opt-in off-box carry
// layer — the plaintext mirror stays the inspectable, sovereign default. The
// secret lives only in the owner's head: no escrow, no cloud, no key-on-disk, so
// a lost passphrase means a lost backup. EncryptDir/DecryptDir are pure and
// tree-shaped — they take a directory path, never a PocketBase app, so they add
// no new collection-read surface beyond the mirror the export already wrote.

// envelope header layout constants. The header is a small plaintext preamble
// that records the KDF salt and GCM nonce; it is also bound as GCM
// additional-authenticated-data, so tampering with any header byte fails the
// auth tag.
const (
	envMagic   = "BALAUREX" // 8 bytes, ASCII envelope identifier
	envVersion = 1          // 1 byte, bumped if the KDF/cipher ever changes
	// envWarning records the unrecoverable property in the envelope itself (the
	// brief's "a flag in the envelope"). It carries no key material.
	envWarning = 1

	scryptN      = 1 << 15 // standard interactive cost (N=32768)
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32 // 32-byte key → AES-256
	saltLen      = 16
)

// ErrBadPassphrase is returned by DecryptDir when authentication fails (wrong
// passphrase or tampered archive). Callers test with errors.Is.
var ErrBadPassphrase = errors.New("export: wrong passphrase or corrupted archive")

// EncryptDir tars srcDir (deterministic file order) and writes a single
// passphrase-encrypted archive to destFile. AES-256-GCM over the tar; the
// per-archive scrypt salt and GCM nonce live in a small plaintext header that
// is also bound as GCM additional-authenticated-data, so tampering with the
// header fails decryption. It never reads any PocketBase collection — it
// operates purely on the already-written directory.
func EncryptDir(srcDir, destFile, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("export: empty passphrase")
	}

	plaintext, err := tarDir(srcDir)
	if err != nil {
		return fmt.Errorf("export: tarring %s: %w", srcDir, err)
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("export: reading salt: %w", err)
	}
	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return fmt.Errorf("export: deriving key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("export: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("export: gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("export: reading nonce: %w", err)
	}

	header := buildHeader(salt, nonce)
	ciphertext := gcm.Seal(nil, nonce, plaintext, header)

	out := make([]byte, 0, len(header)+len(ciphertext))
	out = append(out, header...)
	out = append(out, ciphertext...)
	if err := os.WriteFile(destFile, out, 0o600); err != nil {
		return fmt.Errorf("export: writing %s: %w", destFile, err)
	}
	return nil
}

// DecryptDir reverses EncryptDir: it reads the header, re-derives the key from
// the passphrase, GCM-opens the ciphertext, and untars into destDir. A wrong
// passphrase (or any tampered byte) fails the GCM auth tag and returns
// ErrBadPassphrase with NOTHING written to destDir (decrypt fully in memory,
// untar only after Open succeeds). Exposed as the `balaur restore` CLI verb.
func DecryptDir(srcFile, destDir, passphrase string) error {
	if passphrase == "" {
		return fmt.Errorf("export: empty passphrase")
	}
	blob, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("export: reading %s: %w", srcFile, err)
	}

	header, salt, nonce, ciphertext, err := parseHeader(blob)
	if err != nil {
		return err
	}

	key, err := scrypt.Key([]byte(passphrase), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return fmt.Errorf("export: deriving key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("export: cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("export: gcm: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return ErrBadPassphrase
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, header)
	if err != nil {
		// Never wrap the GCM error — a tampered tag and a wrong passphrase are
		// indistinguishable and neither leaks anything useful.
		return ErrBadPassphrase
	}
	if err := untar(plaintext, destDir); err != nil {
		return fmt.Errorf("export: untarring into %s: %w", destDir, err)
	}
	return nil
}

// buildHeader serializes the plaintext envelope preamble: magic, version, the
// unrecoverable-warning flag, then the length-prefixed salt and nonce. The
// returned slice is passed verbatim as GCM additional-authenticated-data.
func buildHeader(salt, nonce []byte) []byte {
	var b bytes.Buffer
	b.WriteString(envMagic)
	b.WriteByte(envVersion)
	b.WriteByte(envWarning)
	b.WriteByte(byte(len(salt)))
	b.Write(salt)
	b.WriteByte(byte(len(nonce)))
	b.Write(nonce)
	return b.Bytes()
}

// parseHeader splits an envelope into its header (the exact AAD bytes), salt,
// nonce, and trailing ciphertext. A malformed preamble is reported as
// ErrBadPassphrase (it is a corrupted/foreign archive — same blunt outcome).
func parseHeader(blob []byte) (header, salt, nonce, ciphertext []byte, err error) {
	r := bytes.NewReader(blob)
	magic := make([]byte, len(envMagic))
	if _, e := io.ReadFull(r, magic); e != nil || string(magic) != envMagic {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	var version, warning, sLen uint8
	if e := binary.Read(r, binary.BigEndian, &version); e != nil || version != envVersion {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	if e := binary.Read(r, binary.BigEndian, &warning); e != nil {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	if e := binary.Read(r, binary.BigEndian, &sLen); e != nil {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	salt = make([]byte, sLen)
	if _, e := io.ReadFull(r, salt); e != nil {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	var nLen uint8
	if e := binary.Read(r, binary.BigEndian, &nLen); e != nil {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	nonce = make([]byte, nLen)
	if _, e := io.ReadFull(r, nonce); e != nil {
		return nil, nil, nil, nil, ErrBadPassphrase
	}
	headerLen := len(blob) - r.Len()
	header = blob[:headerLen]
	ciphertext = blob[headerLen:]
	return header, salt, nonce, ciphertext, nil
}

// tarDir walks srcDir, collects relative paths in sorted order, and writes each
// regular file into a tar buffer. Sorting makes the tar (and thus, for a fixed
// salt+nonce, the plaintext) deterministic. Mirror trees are small, so buffering
// the whole tar in memory is the simplest correct option.
func tarDir(srcDir string) ([]byte, error) {
	var rels []string
	err := filepath.WalkDir(srcDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, rErr := filepath.Rel(srcDir, p)
		if rErr != nil {
			return rErr
		}
		rels = append(rels, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(rels)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, rel := range rels {
		data, rErr := os.ReadFile(filepath.Join(srcDir, rel))
		if rErr != nil {
			return nil, rErr
		}
		hdr := &tar.Header{
			Name: filepath.ToSlash(rel),
			Mode: 0o644,
			Size: int64(len(data)),
		}
		if wErr := tw.WriteHeader(hdr); wErr != nil {
			return nil, wErr
		}
		if _, wErr := tw.Write(data); wErr != nil {
			return nil, wErr
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// untar reconstructs the tree under destDir from the decrypted tar bytes. It
// guards against path traversal: any entry whose cleaned target escapes destDir
// is rejected before any write.
func untar(plaintext []byte, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	cleanDest := filepath.Clean(destDir)
	tr := tar.NewReader(bytes.NewReader(plaintext))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		target := filepath.Join(cleanDest, filepath.FromSlash(hdr.Name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("export: unsafe tar path %q", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data := make([]byte, hdr.Size)
		if _, err := io.ReadFull(tr, data); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
