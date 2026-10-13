import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

kotlin {
    compilerOptions {
        jvmTarget.set(JvmTarget.JVM_17)
    }
}

dependencies {
    api(libs.kotlinx.coroutines.core)
    api(libs.kotlinx.serialization.json)
    api(libs.okhttp)

    testImplementation(libs.junit)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.kotlinx.coroutines.test)
}

tasks.test {
    useJUnit()
    // Contract tests against a live server run only when NEWAPI_IT_BASE_URL is set.
    systemProperty("newapi.it.baseUrl", System.getenv("NEWAPI_IT_BASE_URL") ?: "")
    systemProperty("newapi.it.username", System.getenv("NEWAPI_IT_USERNAME") ?: "")
    systemProperty("newapi.it.password", System.getenv("NEWAPI_IT_PASSWORD") ?: "")
    testLogging {
        events("passed", "skipped", "failed")
        exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
    }
}
