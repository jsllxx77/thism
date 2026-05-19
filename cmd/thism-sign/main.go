// Command thism-sign provides offline release key management and binary
// signing for the agent self-update channel.
//
// Subcommands:
//
//	thism-sign keygen -out-pub PUB -out-priv PRIV
//	    Generate a new Ed25519 keypair. PUB receives the base64-encoded
//	    public key (suitable for the -X ldflags injection point
//	    github.com/thism-dev/thism/internal/security/release.PublicKeyBase64).
//	    PRIV receives the base64-encoded private key — keep this file
//	    offline.
//
//	thism-sign sign -priv PRIV -in BINARY -out BINARY.sig
//	    Sign a single binary. Output is hex-encoded Ed25519 signature.
//
//	thism-sign sign-dist -priv PRIV [-dir dist]
//	    Sign every thism-agent-* file under -dir, writing
//	    <file>.sig alongside it. Skips files whose name already ends in
//	    .sig or .version.
//
// The private key is loaded from disk (-priv) or from the
// THISM_RELEASE_PRIVATE_KEY environment variable. The signing tool never
// connects to the network and never reads server-side state.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const envPrivateKey = "THISM_RELEASE_PRIVATE_KEY"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "thism-sign:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "keygen":
		return cmdKeygen(args[1:])
	case "sign":
		return cmdSign(args[1:])
	case "sign-dist":
		return cmdSignDist(args[1:])
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  thism-sign keygen -out-pub PUB -out-priv PRIV")
	fmt.Fprintln(w, "  thism-sign sign -priv PRIV -in BINARY -out BINARY.sig")
	fmt.Fprintln(w, "  thism-sign sign-dist -priv PRIV [-dir dist]")
}

func cmdKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	outPub := fs.String("out-pub", "", "path to write base64 public key")
	outPriv := fs.String("out-priv", "", "path to write base64 private key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outPub == "" || *outPriv == "" {
		return errors.New("keygen requires -out-pub and -out-priv")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}
	if err := writeBase64File(*outPub, pub, 0o644); err != nil {
		return fmt.Errorf("write pub: %w", err)
	}
	if err := writeBase64File(*outPriv, priv, 0o600); err != nil {
		return fmt.Errorf("write priv: %w", err)
	}
	fmt.Printf("wrote public key  -> %s\n", *outPub)
	fmt.Printf("wrote private key -> %s (mode 0600, keep offline)\n", *outPriv)
	return nil
}

func cmdSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	privPath := fs.String("priv", "", "path to base64 private key (or use $"+envPrivateKey+")")
	in := fs.String("in", "", "binary file to sign")
	out := fs.String("out", "", "output signature file (hex)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" || *out == "" {
		return errors.New("sign requires -in and -out")
	}
	priv, err := loadPrivateKey(*privPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	sig := ed25519.Sign(priv, data)
	if err := os.WriteFile(*out, []byte(hex.EncodeToString(sig)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write signature: %w", err)
	}
	fmt.Printf("signed %s -> %s\n", *in, *out)
	return nil
}

func cmdSignDist(args []string) error {
	fs := flag.NewFlagSet("sign-dist", flag.ContinueOnError)
	privPath := fs.String("priv", "", "path to base64 private key (or use $"+envPrivateKey+")")
	dir := fs.String("dir", "dist", "directory containing thism-agent-* binaries")
	if err := fs.Parse(args); err != nil {
		return err
	}
	priv, err := loadPrivateKey(*privPath)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(*dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	signed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "thism-agent-") {
			continue
		}
		if strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".version") {
			continue
		}
		path := filepath.Join(*dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		sig := ed25519.Sign(priv, data)
		sigPath := path + ".sig"
		if err := os.WriteFile(sigPath, []byte(hex.EncodeToString(sig)+"\n"), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", sigPath, err)
		}
		fmt.Printf("signed %s -> %s\n", path, sigPath)
		signed++
	}
	if signed == 0 {
		return fmt.Errorf("no thism-agent-* binaries found in %s", *dir)
	}
	return nil
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	var b64 string
	switch {
	case path != "":
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read private key: %w", err)
		}
		b64 = strings.TrimSpace(string(raw))
	case os.Getenv(envPrivateKey) != "":
		b64 = strings.TrimSpace(os.Getenv(envPrivateKey))
	default:
		return nil, fmt.Errorf("private key required (use -priv or $%s)", envPrivateKey)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode private key base64: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("expected %d byte private key, got %d", ed25519.PrivateKeySize, len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}

func writeBase64File(path string, data []byte, mode os.FileMode) error {
	encoded := base64.StdEncoding.EncodeToString(data) + "\n"
	return os.WriteFile(path, []byte(encoded), mode)
}
