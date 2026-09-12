"""Regressions for the concrete command shapes reported by catalog PR #1978."""
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
import verify_skill


class ScopedCommandContracts(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.cli = Path(self.temp.name)
        self.source = self.cli / "internal" / "cli"
        self.source.mkdir(parents=True)

    def write(self, source):
        (self.source / "root.go").write_text(source, encoding="utf-8")

    def test_local_string_flags_shadow_global_and_same_file_booleans(self):
        self.write('''package cli
func Execute() {
 rootCmd := &cobra.Command{Use: "demo-pp-cli"}
 rootCmd.PersistentFlags().BoolVar(&csv, "csv", false, "CSV output")
 rootCmd.AddCommand(newBatchCmd())
 rootCmd.AddCommand(newOtherCmd())
}
func newBatchCmd() *cobra.Command {
 cmd := &cobra.Command{Use: "batch <id>"}
 cmd.Flags().StringVar(&csv, "csv", "", "Input CSV")
 cmd.Flags().StringVar(&title, "title", "", "Title")
 cmd.Flags().StringVar(&active, "active", "", "True, Any, False")
 cmd.Flags().BoolVar(&json, "json", false, "JSON output")
 return cmd
}
func newOtherCmd() *cobra.Command {
 cmd := &cobra.Command{Use: "other"}
 cmd.Flags().BoolVar(&title, "title", false, "Title only")
 cmd.Flags().BoolVar(&active, "active", false, "Active only")
 return cmd
}
''')
        path, positional, flags = verify_skill._cli_invocation_from_tokens(
            ["batch", "--csv", "contacts.csv", "--title", "Sample", "--active", "True", "--json", "42"], self.cli)
        self.assertEqual(path, ["batch"])
        self.assertEqual(positional, ["42"])
        self.assertEqual(flags, ["--csv", "--title", "--active", "--json"])
        # A sibling's local flag remains boolean and cannot hide a stray value.
        self.assertEqual(verify_skill._cli_invocation_from_tokens(
            ["other", "--active", "false"], self.cli)[1], ["false"])

    def test_option_separator_preserves_negative_and_flag_shaped_positionals(self):
        self.assertEqual(verify_skill._cli_invocation_from_tokens(
            ["point", "--elev", "8530", "--", "-111.5838,40.5884", "--literal"], None),
            (["point"], ["-111.5838,40.5884", "--literal"], ["--elev"]))

    def test_constructor_variables_attach_only_to_the_returned_parent(self):
        self.write('''package cli
func Execute() { rootCmd.AddCommand(newUsersCmd()) }
func newUsersCmd() *cobra.Command {
 cmd := &cobra.Command{Use: "users <user_id>"}
 { sub := newItemsCmd(); cmd.AddCommand(sub) }
 { sub := newCollectionsCmd(); cmd.AddCommand(sub) }
 orphan := newOrphanCmd()
 unrelated.AddCommand(orphan)
 return cmd
}
func newItemsCmd() *cobra.Command { return &cobra.Command{Use: "items"} }
func newCollectionsCmd() *cobra.Command { return &cobra.Command{Use: "collections"} }
func newOrphanCmd() *cobra.Command { return &cobra.Command{Use: "orphan"} }
''')
        for name in ("items", "collections"):
            self.assertIsNotNone(verify_skill.resolve_command_path(self.cli, ["users", name])[0])
        self.assertIsNone(verify_skill.resolve_command_path(self.cli, ["users", "orphan"])[0])

    def test_find_attachment_requires_existing_parent_and_matching_receiver(self):
        self.write('''package cli
func Execute() {
 rootCmd.AddCommand(newUsersCmd())
 if bm, _, err := rootCmd.Find([]string{"users", "bookmarks"}); err == nil && bm.Name() == "bookmarks" {
  bm.AddCommand(newNovelFindCmd())
  other.AddCommand(newWrongCmd())
 }
 if missing, _, err := rootCmd.Find([]string{"absent"}); err == nil {
  missing.AddCommand(newOrphanCmd())
 }
}
func newUsersCmd() *cobra.Command {
 cmd := &cobra.Command{Use: "users"}
 cmd.AddCommand(newBookmarksCmd())
 return cmd
}
func newBookmarksCmd() *cobra.Command { return &cobra.Command{Use: "bookmarks"} }
func newNovelFindCmd() *cobra.Command { return &cobra.Command{Use: "find [query]"} }
func newWrongCmd() *cobra.Command { return &cobra.Command{Use: "wrong"} }
func newOrphanCmd() *cobra.Command { return &cobra.Command{Use: "orphan"} }
''')
        self.assertEqual(verify_skill.resolve_command_path(self.cli, ["users", "bookmarks", "find"])[1], "find [query]")
        for path in (["users", "bookmarks", "wrong"], ["absent", "orphan"], ["find"]):
            self.assertIsNone(verify_skill.resolve_command_path(self.cli, path)[0])


if __name__ == "__main__":
    unittest.main()
