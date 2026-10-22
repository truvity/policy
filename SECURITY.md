# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately via
[GitHub Security Advisories](https://github.com/truvity/policy/security/advisories/new).

Do NOT open a public issue for security vulnerabilities.

## Supported Versions

Only the latest release is supported with security updates.

## What is in scope

This repository publishes contracts, schemas, the Go configuration loader and
conformance helpers, and the library chart. The worked example and the
TypeScript, Python and Kotlin loaders are in the
[url-shortener repository](https://github.com/truvity/url-shortener); report anything about them there. Reports
that matter most:

- A loader that accepts a configuration it should refuse, or that reports a
  secret's value in an error or a log line.
- A contract or a schema whose defaults are unsafe for anyone who follows
  them.
- Anything in the guides or the library chart that would be a vulnerability
  in a real service, since they are what people copy.

This repository holds no credentials and its CI runs on hosted runners with
no access to any private infrastructure. A finding that depends on a
particular deployment belongs with that deployment's owner.
