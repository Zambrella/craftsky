import os
import unittest
from unittest import mock

import cloudflare_pages_build


class CloudflarePagesBuildTests(unittest.TestCase):
    def test_preview_branch_does_not_publish_gate(self):
        with (
            mock.patch.dict(
                os.environ,
                {"CF_PAGES_BRANCH": "feature", "CF_PAGES_PRODUCTION_BRANCH": "main"},
                clear=True,
            ),
            mock.patch("cloudflare_pages_build.subprocess.run") as run,
        ):
            self.assertEqual(cloudflare_pages_build.main(), 0)
            run.assert_not_called()

    def test_production_branch_propagates_readiness_failure(self):
        completed = mock.Mock(returncode=1)
        with (
            mock.patch.dict(
                os.environ,
                {"CF_PAGES_BRANCH": "main", "CF_PAGES_PRODUCTION_BRANCH": "main"},
                clear=True,
            ),
            mock.patch(
                "cloudflare_pages_build.subprocess.run", return_value=completed
            ) as run,
        ):
            self.assertEqual(cloudflare_pages_build.main(), 1)
            run.assert_called_once()


if __name__ == "__main__":
    unittest.main()
