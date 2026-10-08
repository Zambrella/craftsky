import os
import unittest
from unittest import mock

import cloudflare_pages_build


class CloudflarePagesBuildTests(unittest.TestCase):
    def test_preview_branch_builds(self):
        with mock.patch.dict(os.environ, {"CF_PAGES_BRANCH": "feature", "CF_PAGES_PRODUCTION_BRANCH": "main"}, clear=True):
            self.assertEqual(cloudflare_pages_build.main(), 0)

    def test_production_branch_builds_with_open_readiness_gaps(self):
        with mock.patch.dict(os.environ, {"CF_PAGES_BRANCH": "main", "CF_PAGES_PRODUCTION_BRANCH": "main"}, clear=True):
            self.assertEqual(cloudflare_pages_build.main(), 0)


if __name__ == "__main__":
    unittest.main()
