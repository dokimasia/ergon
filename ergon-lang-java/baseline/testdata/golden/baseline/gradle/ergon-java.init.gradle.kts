// Managed by ergon init. Add repository settings to .ergon/local/gradle/ergon-java.init.gradle.kts and run ergon init sync.
//
// The lint of the Java sources, which lint-java applies to every project of the Gradle build with
// --init-script, so the build files of the repository stay as they are. javac and javadoc report
// every warning as an error, and PMD checks the sources with its base ruleset. PMD's own guide
// advises against every rule of every category, which reports mostly unimportant findings.

allprojects {
    tasks.withType<JavaCompile>().configureEach {
        options.compilerArgs.addAll(listOf("-Xlint:all", "-Werror"))
    }
    tasks.withType<Javadoc>().configureEach {
        (options as StandardJavadocDocletOptions).apply {
            addStringOption("Xdoclint:all", "-quiet")
            addBooleanOption("Werror", true)
        }
    }
    pluginManager.withPlugin("java") {
        apply(plugin = "pmd")
        extensions.configure<PmdExtension> {
            toolVersion = "7.28.0"
            isConsoleOutput = true
            ruleSets = listOf("rulesets/java/quickstart.xml")
        }
    }
}
