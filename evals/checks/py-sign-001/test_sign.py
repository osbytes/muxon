import unittest

from sign import sign


class SignTest(unittest.TestCase):
    def test_sign(self):
        self.assertEqual(sign(3), 1)
        self.assertEqual(sign(-2), -1)
        self.assertEqual(sign(0), 0)
