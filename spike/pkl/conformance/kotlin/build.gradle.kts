// The Kotlin side of the conformance run: the classes pkl-codegen-kotlin
// generates from the contract, loaded with pkl-config-kotlin, and networknt
// (the library the repository's own Kotlin loader uses) on the JSON Schemas.
plugins {
    kotlin("jvm") version "2.4.20"
    application
}

repositories { mavenCentral() }

kotlin { jvmToolchain(21) }

sourceSets.main {
    kotlin.srcDir("src/main/kotlin/gen/kotlin")
    resources.srcDir("src/main/kotlin/gen/resources")
}

dependencies {
    implementation("org.pkl-lang:pkl-config-kotlin:0.31.1")
    implementation("org.pkl-lang:pkl-config-java:0.31.1")
    implementation("org.pkl-lang:pkl-core:0.31.1")
    implementation("com.fasterxml.jackson.core:jackson-databind:2.22.3")
    implementation("com.fasterxml.jackson.dataformat:jackson-dataformat-yaml:2.22.3")
    implementation("com.fasterxml.jackson.module:jackson-module-kotlin:2.22.3")
    implementation("com.networknt:json-schema-validator:1.5.9")
}

application { mainClass.set("MainKt") }
