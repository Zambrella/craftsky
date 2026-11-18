# Restricted evidence storage

Status, 9 November 2026: Douglas Todd approved provisioning in the existing AWS
account. The bucket and dedicated IAM user/policy were created through the
signed-in CraftSky console and their saved settings inspected. AppView is not
connected yet; no access key has been created and no Render configuration or
deployment has been performed for this storage.

## Provisioning evidence

- Bucket: `craftsky-prod-safety-evidence-17482696`, created 9 November 2026 at
  13:30 UTC, in Frankfurt (`eu-central-1`). Console showed zero objects.
- Verified Block all public access On, Bucket owner enforced, default SSE-S3,
  versioning Disabled, Object Lock Disabled and no lifecycle rules.
- Saved the HTTPS-only bucket policy below and reopened its editor to verify
  the persisted policy and exact bucket resources.
- Created IAM user `craftsky-appview-safety-evidence`, console access Disabled,
  with one inline policy `CraftSkyRestrictedEvidenceOnly`. Expanded the saved
  policy and verified it matches the bucket-only actions below.
- No access key, synthetic object, hold/deletion exercise or effective
  application-access test has been performed. These remain outstanding.

## Production settings

The dedicated S3 bucket in `eu-central-1` is separate from scheduled-post media.
Preserve the following settings:

- Enable all four Block Public Access settings.
- Use Bucket owner enforced ownership with ACLs disabled.
- Enable default SSE-S3 encryption.
- Leave versioning disabled; do not enable Object Lock.
- Do not configure lifecycle expiration. AppView must check database legal
  holds before deleting evidence.
- Require HTTPS using the bucket policy below. Do not attach a public CDN.
- Create a dedicated AppView IAM principal with the policy below; do not reuse
  scheduled-media credentials or grant access to other buckets.

The current store uses `HeadBucket`, `PutObject`, `GetObject` and `DeleteObject`.
Its delete request does not specify a version ID. AWS documents that deletion
from a versioned bucket leaves previous object versions behind:
https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html

Application legal holds are not S3 Object Lock and do not prevent an AWS
administrator from deleting objects. Administrative access must remain limited.

## Dedicated AppView IAM policy

Replace `BUCKET_NAME` with the confirmed bucket name before applying:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "InspectRestrictedEvidenceBucket",
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::BUCKET_NAME"
    },
    {
      "Sid": "ManageRestrictedEvidenceObjects",
      "Effect": "Allow",
      "Action": ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::BUCKET_NAME/*"
    }
  ]
}
```

## Bucket transport policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyInsecureTransport",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": ["arn:aws:s3:::BUCKET_NAME", "arn:aws:s3:::BUCKET_NAME/*"],
      "Condition": {"Bool": {"aws:SecureTransport": "false"}}
    }
  ]
}
```

## AppView configuration and verification

Configure these values through the approved production secret/configuration
path, without recording credentials in chat, logs or repository files:

```text
SAFETY_EVIDENCE_S3_ENDPOINT=https://s3.eu-central-1.amazonaws.com
SAFETY_EVIDENCE_S3_REGION=eu-central-1
SAFETY_EVIDENCE_S3_BUCKET=<confirmed dedicated bucket>
SAFETY_EVIDENCE_S3_ACCESS_KEY_ID=<dedicated principal access key>
SAFETY_EVIDENCE_S3_SECRET_ACCESS_KEY=<dedicated principal secret>
```

Before activation, verify bucket ownership, public-access blocking, encryption,
versioning, absence of lifecycle expiration and the dedicated principal's
permissions. Configuration and deployment remain separate production changes.

Use only benign synthetic evidence for the operational exercise:

1. Confirm AppView starts and its evidence-store readiness check succeeds.
2. Attach synthetic evidence using the authorized console path and verify
   authorized retrieval and audit history; confirm anonymous retrieval fails.
3. Place an application legal hold and verify retention/deletion preserves the
   held evidence through the intended workflow.
4. Release the synthetic hold when appropriate, exercise the applicable deletion
   workflow, and verify the object is removed and the action is audited.

Record actual results and remaining limitations before closing CSEA-02 or the
preservation dependency under LE-01. Never upload actual illegal material as a
test. This setup does not itself approve a retention period or a disclosure.
