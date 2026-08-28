#!/usr/bin/env python3
"""
Secret & PII scanner — the enforcement behind the "nothing leaks to GitHub" rule.

Used by git hooks (pre-commit / pre-push) and CI.

Modes:
    --staged            scan staged changes            (pre-commit)
    --range A..B        scan commits about to be sent  (pre-push)
    --files F [F...]    scan specific files
    --tracked           scan every tracked file        (audit)
    --history           scan all of history            (audit; slow)

Exit codes: 0 clean · 1 findings · 2 usage/environment error.

Suppressing a false positive:
  - inline, on the offending line:  allowlist-secret
  - repo-wide, in .secretscanignore: one regex or path glob per line (# comments)

Repository-specific forbidden artifact types belong in .secretscanpolicy.
The scanner blocks publication; it never edits or cleans source files.
"""

from __future__ import annotations

import argparse
import fnmatch
import os
import re
import subprocess
import sys
from dataclasses import dataclass

# --- rules ------------------------------------------------------------------
# Ordered most-specific first. `severity` separates "this is definitely a
# credential" from "this looks like personal data" so the report can lead with
# what matters, but BOTH block: a leak is a leak.


@dataclass(frozen=True)
class Rule:
    name: str
    pattern: re.Pattern
    kind: str  # "secret" | "pii"


RULES: list[Rule] = [
    # -- credentials: high-confidence, provider-specific shapes --
    Rule("private-key-block", re.compile(r"-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----"), "secret"),
    Rule("aws-access-key-id", re.compile(r"\b(?:AKIA|ASIA)[0-9A-Z]{16}\b"), "secret"),
    Rule("aws-secret-access-key", re.compile(r"(?i)aws.{0,20}secret.{0,20}['\"][0-9a-zA-Z/+]{40}['\"]"), "secret"),
    Rule("github-token", re.compile(r"\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b"), "secret"),
    Rule("github-fine-grained-pat", re.compile(r"\bgithub_pat_[A-Za-z0-9_]{22,}\b"), "secret"),
    Rule("anthropic-api-key", re.compile(r"\bsk-ant-[A-Za-z0-9\-_]{20,}\b"), "secret"),
    Rule("openai-api-key", re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9]{32,}\b"), "secret"),
    Rule("api-key-prefix", re.compile(r"\bAIza[0-9A-Za-z\-_]{35}\b"), "secret"),
    Rule("slack-token", re.compile(r"\bxox[baprs]-[A-Za-z0-9\-]{10,}\b"), "secret"),
    Rule("stripe-live-key", re.compile(r"\b(?:sk|rk)_live_[A-Za-z0-9]{20,}\b"), "secret"),
    Rule("npm-token", re.compile(r"\bnpm_[A-Za-z0-9]{36}\b"), "secret"),
    Rule("jwt", re.compile(r"\bey[A-Za-z0-9_-]{10,}\.ey[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b"), "secret"),
    Rule("basic-auth-in-url", re.compile(r"://[^/\s:@]+:[^/\s:@]{3,}@[^/\s]+"), "secret"),
    # -- credentials: generic assignments. Value must look real, not a placeholder. --
    Rule(
        "generic-credential-assignment",
        re.compile(
            r"(?i)\b(?:api[_-]?key|secret[_-]?key|access[_-]?token|auth[_-]?token|client[_-]?secret|password|passwd)\b"
            r"\s*[:=]\s*['\"]([^'\"\s]{8,})['\"]"
        ),
        "secret",
    ),
    # -- personal data --
    Rule("email-address", re.compile(r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b"), "pii"),
    Rule("credit-card", re.compile(r"\b(?:\d[ -]*?){13,19}\b"), "pii"),
    Rule("us-ssn", re.compile(r"\b(?!000|666|9\d\d)\d{3}-(?!00)\d{2}-(?!0000)\d{4}\b"), "pii"),
    Rule("iban", re.compile(r"\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b"), "pii"),
    Rule("phone-e164", re.compile(r"(?<![\w.])\+\d{1,3}[ -]?\(?\d{2,4}\)?[ -]?\d{3}[ -]?\d{3,4}(?![\w.])"), "pii"),
]

# Values that match a rule's shape but carry no real secret. Kept deliberately
# tight: a broad allowlist is how these scanners end up useless.
PLACEHOLDER_VALUES = re.compile(
    r"(?i)^(?:x{3,}|\*{3,}|\.{3,}|<[^>]+>|\$\{[^}]+\}|\{\{[^}]+\}\}|%s|None|null|nil|true|false|"
    r"your[_-].*|my[_-].*|example.*|sample.*|dummy.*|placeholder.*|changeme.*|redacted.*|secret|password|"
    r"test|testing|foo|bar|baz|todo|tbd|n/?a)$"
)

# Domains that are reserved for documentation or are non-identifying.
SAFE_EMAIL_DOMAINS = (
    "example.com", "example.org", "example.net", "test.com", "localhost",
    "sentry.io", "email.com",
)
SAFE_EMAIL_LOCALPARTS = ("noreply", "no-reply", "you", "your", "user", "someone", "me", "test", "email")

# Binary/vendored paths where scanning produces noise, not signal.
SKIP_PATH_GLOBS = (
    "*/node_modules/*", "node_modules/*", "*/vendor/*", "vendor/*", "*/.git/*",
    "*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.ico", "*.pdf", "*.zip",
    "*.gz", "*.tar", "*.woff", "*.woff2", "*.ttf", "*.eot", "*.mp4", "*.mov",
    "*.wasm", "*.so", "*.dylib", "*.dll", "*.class", "*.jar",
    "package-lock.json", "*/package-lock.json",
    "*.min.js", "*.min.css", "*.map",
)

# Files whose very presence in a commit is the problem.
FORBIDDEN_FILE_GLOBS = (
    ".env", "*/.env", ".env.*", "*/.env.*",
    "*.pem", "*.key", "*.p12", "*.pfx", "*.keystore", "*.jks",
    "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "*/id_rsa", "*/id_ed25519",
    ".npmrc", "*/.npmrc", ".pypirc", ".netrc", "*/.netrc",
    ".aws/credentials", "*/.aws/credentials", "credentials.json", "*/credentials.json",
    "client_secret*.json", "*/client_secret*.json", "token.json", "*/token.json",
    "tokens.json", "*/tokens.json",
    "service-account*.json", "*/service-account*.json",
)
FORBIDDEN_FILE_EXCEPTIONS = (".env.example", ".env.sample", ".env.template", "*/.env.example", "*/.env.sample")

INLINE_ALLOW = "allowlist-secret"
POLICY_FILE = ".secretscanpolicy"


@dataclass
class Finding:
    path: str
    line_no: int
    rule: str
    kind: str
    excerpt: str


def sh(args: list[str]) -> str:
    return subprocess.run(args, capture_output=True, text=True, check=False).stdout


def load_repo_allowlist(repo_root: str) -> list[str]:
    path = os.path.join(repo_root, ".secretscanignore")
    if not os.path.isfile(path):
        return []
    out = []
    with open(path, "r", errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if line and not line.startswith("#"):
                out.append(line)
    return out


def load_path_policy(repo_root: str) -> list[str]:
    """Load ordered forbidden-path rules. A leading ! allows a narrow path."""
    path = os.path.join(repo_root, POLICY_FILE)
    if not os.path.isfile(path):
        return []
    out = []
    with open(path, "r", errors="replace") as fh:
        for line in fh:
            line = line.strip()
            if line and not line.startswith("#") and line != "!":
                out.append(line)
    return out


def allowlisted(text: str, path: str, patterns: list[str]) -> bool:
    for p in patterns:
        if fnmatch.fnmatch(path, p):
            return True
        try:
            if re.search(p, text):
                return True
        except re.error:
            pass  # a malformed user regex must not break the gate
    return False


def skip_path(path: str) -> bool:
    return any(fnmatch.fnmatch(path, g) for g in SKIP_PATH_GLOBS)


def policy_forbids(path: str, rules: list[str]) -> bool:
    """Apply ordered path rules. The last matching forbid or allow rule wins."""
    forbidden = False
    for rule in rules:
        allowed = rule.startswith("!")
        pattern = rule[1:] if allowed else rule
        if fnmatch.fnmatch(path, pattern):
            forbidden = not allowed
    return forbidden


def luhn_ok(digits: str) -> bool:
    ds = [int(c) for c in digits if c.isdigit()]
    if not 13 <= len(ds) <= 19:
        return False
    total, alt = 0, False
    for d in reversed(ds):
        if alt:
            d *= 2
            if d > 9:
                d -= 9
        total += d
        alt = not alt
    return total % 10 == 0


def redact(s: str) -> str:
    s = s.strip()
    if len(s) <= 12:
        return s[:3] + "***"
    return f"{s[:6]}…{s[-4:]} ({len(s)} chars)"


def is_false_positive(rule: Rule, match: re.Match, line: str) -> bool:
    raw = match.group(0)

    if rule.name == "generic-credential-assignment":
        value = match.group(1)
        if PLACEHOLDER_VALUES.match(value):
            return True
        # Interpolations and obvious references aren't literals.
        if any(t in value for t in ("${", "{{", "process.env", "os.environ", "$(", "%(")):
            return True
        # Require some variety; "aaaaaaaa" or "--------" isn't a credential.
        if len(set(value)) < 5:
            return True

    elif rule.name == "email-address":
        local, _, domain = raw.partition("@")
        if domain.lower() in SAFE_EMAIL_DOMAINS:
            return True
        if local.lower() in SAFE_EMAIL_LOCALPARTS:
            return True
        # Emails in package metadata/authorship are intentional, not leaks.
        if re.search(r"(?i)\b(?:author|maintainer|copyright|contributor)\b", line):
            return True

    elif rule.name == "credit-card":
        if not luhn_ok(raw):
            return True

    elif rule.name == "iban":
        # Avoid flagging ordinary uppercase identifiers/hashes.
        if not re.match(r"^[A-Z]{2}\d{2}", raw) or raw.isalpha():
            return True

    elif rule.name == "jwt":
        if PLACEHOLDER_VALUES.match(raw):
            return True

    return False


def scan_text(path: str, text: str, allowlist: list[str]) -> list[Finding]:
    findings: list[Finding] = []
    for i, line in enumerate(text.splitlines(), start=1):
        if INLINE_ALLOW in line:
            continue
        for rule in RULES:
            for m in rule.pattern.finditer(line):
                if is_false_positive(rule, m, line):
                    continue
                if allowlisted(m.group(0), path, allowlist):
                    continue
                findings.append(Finding(path, i, rule.name, rule.kind, redact(m.group(0))))
    return findings


def check_forbidden_files(paths: list[str], policy: list[str] | None = None) -> list[Finding]:
    policy = policy or []
    out = []
    for p in paths:
        if any(fnmatch.fnmatch(p, g) for g in FORBIDDEN_FILE_EXCEPTIONS):
            continue
        if any(fnmatch.fnmatch(p, g) for g in FORBIDDEN_FILE_GLOBS):
            out.append(Finding(p, 0, "forbidden-file", "secret", "file must never be committed"))
        elif policy_forbids(p, policy):
            out.append(Finding(p, 0, "policy-forbidden-file", "sensitive", "file type is blocked by repository policy"))
    return out


def scan_diff(diff: str, allowlist: list[str]) -> list[Finding]:
    """Scan added lines of a unified diff, tracking the file and line number."""
    findings: list[Finding] = []
    path = "?"
    new_line_no = 0
    for line in diff.splitlines():
        if line.startswith("+++ b/"):
            # Git appends a tab and metadata after the filename when the path
            # contains a space ("+++ b/a file.txt\t"). Keeping the tab makes
            # every path glob — .secretscanignore entries and SKIP_PATH_GLOBS
            # alike — silently fail to match for exactly those files.
            path = line[6:].split("\t", 1)[0]
            continue
        if line.startswith("@@"):
            m = re.search(r"\+(\d+)", line)
            new_line_no = int(m.group(1)) if m else 0
            continue
        if line.startswith("+") and not line.startswith("+++"):
            if not skip_path(path):
                added = scan_text(path, line[1:], allowlist)
                for f in added:
                    f.line_no = new_line_no
                findings.extend(added)
            new_line_no += 1
        elif not line.startswith("-"):
            new_line_no += 1
    return findings


def report(findings: list[Finding]) -> None:
    secrets = [f for f in findings if f.kind == "secret"]
    pii = [f for f in findings if f.kind == "pii"]
    sensitive = [f for f in findings if f.kind == "sensitive"]

    print("\n\033[31m✗ BLOCKED — potential secrets or personal data detected\033[0m\n", file=sys.stderr)
    for title, group in (("Credentials", secrets), ("Personal data", pii), ("Blocked artifacts", sensitive)):
        if not group:
            continue
        print(f"  {title}:", file=sys.stderr)
        for f in group:
            loc = f"{f.path}:{f.line_no}" if f.line_no else f.path
            print(f"    {loc}\n      {f.rule}: {f.excerpt}", file=sys.stderr)
        print("", file=sys.stderr)

    print(
        "  Fix, don't bypass:\n"
        "    • Real credential? Remove it, then ROTATE it — assume it is compromised.\n"
        "    • Move it to an env var or an untracked .env file.\n"
        "    • False positive? Add `allowlist-secret` on the line, or a pattern to\n"
        "      .secretscanignore — and say so out loud rather than silently.\n",
        file=sys.stderr,
    )


def changed_paths(revision: str) -> list[str]:
    """List paths introduced by a range or by one root/single commit."""
    if ".." in revision:
        args = ["git", "diff", "--name-only", "--diff-filter=ACMR", revision]
    else:
        args = [
            "git", "diff-tree", "--root", "--no-commit-id", "--name-only",
            "--diff-filter=ACMR", "-r", revision,
        ]
    return [p for p in sh(args).splitlines() if p]


def revision_diff(revision: str) -> str:
    """Return added text for a range or for one root/single commit."""
    if ".." in revision:
        return sh(["git", "diff", "--unified=0", revision])
    return sh(["git", "show", "--unified=0", "--format=", revision])


def scan_commit_metadata(revision: str, allowlist: list[str]) -> list[Finding]:
    """Scan author, committer and message metadata exposed by public history."""
    metadata = sh([
        "git", "show", "-s", "--format=author: %an <%ae>%ncommitter: %cn <%ce>%n%n%B",
        revision,
    ])
    return scan_text(f"commit:{revision}", metadata, allowlist)


def policy_path(repo_root: str, path: str) -> str:
    """Make explicit paths comparable with repository-relative policy globs."""
    if not os.path.isabs(path):
        return path.removeprefix("./")
    try:
        rel = os.path.relpath(path, repo_root)
    except ValueError:
        return path
    return rel if rel != ".." and not rel.startswith(f"..{os.sep}") else path


def main() -> int:
    ap = argparse.ArgumentParser(description="Scan for secrets and PII before anything reaches a remote.")
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--staged", action="store_true")
    g.add_argument("--range", metavar="A..B")
    g.add_argument("--files", nargs="+")
    g.add_argument("--tracked", action="store_true")
    g.add_argument("--history", action="store_true")
    ap.add_argument("--quiet", action="store_true", help="print nothing when clean")
    args = ap.parse_args()

    repo_root = sh(["git", "rev-parse", "--show-toplevel"]).strip()
    if not repo_root:
        if args.files:
            repo_root = os.getcwd()
        else:
            print("secret-scan: not inside a git repository", file=sys.stderr)
            return 2

    allowlist = load_repo_allowlist(repo_root)
    policy = load_path_policy(repo_root)
    findings: list[Finding] = []
    scanned = ""

    if args.staged:
        paths = [p for p in sh(["git", "diff", "--cached", "--name-only", "--diff-filter=ACMR"]).splitlines() if p]
        findings += check_forbidden_files(paths, policy)
        findings += scan_diff(sh(["git", "diff", "--cached", "--unified=0"]), allowlist)
        scanned = f"{len(paths)} staged file(s)"

    elif args.range:
        paths = changed_paths(args.range)
        findings += check_forbidden_files(paths, policy)
        findings += scan_diff(revision_diff(args.range), allowlist)
        revs = [r for r in sh(["git", "rev-list", args.range]).splitlines() if r]
        for rev in revs:
            findings += scan_commit_metadata(rev, allowlist)
        scanned = f"{len(paths)} file(s) in {args.range}"

    elif args.files:
        findings += check_forbidden_files([policy_path(repo_root, p) for p in args.files], policy)
        for p in args.files:
            if skip_path(p) or not os.path.isfile(p):
                continue
            with open(p, "r", errors="replace") as fh:
                findings += scan_text(p, fh.read(), allowlist)
        scanned = f"{len(args.files)} file(s)"

    elif args.tracked:
        paths = [p for p in sh(["git", "ls-files"]).splitlines() if p]
        findings += check_forbidden_files(paths, policy)
        for p in paths:
            full = os.path.join(repo_root, p)
            if skip_path(p) or not os.path.isfile(full):
                continue
            with open(full, "r", errors="replace") as fh:
                findings += scan_text(p, fh.read(), allowlist)
        scanned = f"{len(paths)} tracked file(s)"

    elif args.history:
        revs = [r for r in sh(["git", "rev-list", "--all"]).splitlines() if r]
        for rev in revs:
            findings += check_forbidden_files(changed_paths(rev), policy)
            findings += scan_diff(revision_diff(rev), allowlist)
            findings += scan_commit_metadata(rev, allowlist)
        scanned = f"{len(revs)} commit(s) of history"

    # Deduplicate: the same literal repeated is one problem, not twenty.
    seen, unique = set(), []
    for f in findings:
        key = (f.path, f.rule, f.excerpt)
        if key not in seen:
            seen.add(key)
            unique.append(f)

    if unique:
        report(unique)
        return 1

    if not args.quiet:
        print(f"\033[32m✓\033[0m secret-scan: clean ({scanned})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
