# Provider releases

The release workflow imports the shared CTG provider signing key from the organization Actions secrets `TERRAFORM_PROVIDER_GPG_PRIVATE_KEY` and `TERRAFORM_PROVIDER_GPG_PASSPHRASE`. Their selected repository scope and the key pair are managed by `infra-engineering` in the `github/organizations` unit.

After applying that unit, export the public key with `terragrunt output -raw terraform_provider_signing_public_key` and view its fingerprint with `terragrunt output -raw terraform_provider_signing_fingerprint`. Add the public key under Terraform Registry → User Settings → Signing Keys before publishing. Keep previous public keys registered to verify older releases.

Merge the provider changes, then publish a `v2.3.0` tag to run the signed release workflow.
