# Managed by ergon init. Add repository settings to .ergon/local/.tflint.hcl and run ergon init sync.
#
# The configuration of tflint for every module of the repository: every rule of the terraform
# ruleset, which tflint bundles. lint-terraform runs the release of tflint of the section terraform
# of .ergon.yaml. A repository adds the ruleset of its provider, such as tflint-ruleset-aws, in its
# local file.

plugin "terraform" {
  enabled = true
  preset  = "all"
}
