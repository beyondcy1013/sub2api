plugins {
    id("com.android.application")
}

val sub2apiVersion = providers.gradleProperty("sub2apiVersion").orNull
    ?: error("Build with -Psub2apiVersion=<backend version>")
val sub2apiVersionCode = providers.gradleProperty("sub2apiVersionCode").orNull?.toIntOrNull()
    ?: error("Build with -Psub2apiVersionCode=<positive integer>")

android {
    namespace = "com.sub2api.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.sub2api.app"
        minSdk = 24
        targetSdk = 35
        versionCode = sub2apiVersionCode
        versionName = sub2apiVersion
    }

    buildTypes {
        getByName("release") {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    implementation("androidx.core:core:1.15.0")
}
