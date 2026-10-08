
##@ C#
#
# The section csharp of .ergon.yaml sets the options of the C# targets and the steps of
# check-csharp. One run overrides an option on the command line, such as
# make audit-csharp CSHARP_AUDIT_SEVERITY=high.

# The options of the section csharp.
CSHARP_TEST_ARGS ?=
CSHARP_AUDIT_SEVERITY ?= low

.PHONY: fmt-csharp lint-csharp test-csharp audit-csharp check-csharp
fmt: fmt-csharp
lint: lint-csharp
test: test-csharp
audit: audit-csharp
check: check-csharp

fmt-csharp: ## Format the C# sources
	dotnet format
lint-csharp: ## Check the C# sources with every analyzer rule as an error
	dotnet format --verify-no-changes --severity info
	dotnet build --no-incremental -warnaserror -p:TreatWarningsAsErrors=true -p:AnalysisLevel=latest-all \
		-p:AnalysisModeStyle=All -p:EnforceCodeStyleInBuild=true -p:Nullable=enable \
		-p:GenerateDocumentationFile=true -p:WarningLevel=9999
test-csharp: ## Run the C# tests
	dotnet test $(CSHARP_TEST_ARGS)
audit-csharp: ## Scan the NuGet packages for known vulnerabilities
	dotnet restore --force -p:NuGetAudit=true -p:NuGetAuditMode=all -p:NuGetAuditLevel=$(CSHARP_AUDIT_SEVERITY) \
		-p:WarningsAsErrors=NU1900%3BNU1901%3BNU1902%3BNU1903%3BNU1904
check-csharp: lint-csharp test-csharp audit-csharp ## Run the gate of C#
