# R8 rules for the New API Admin app. Library consumer rules (kotlinx.serialization, coroutines,
# OkHttp/Okio, Compose, AndroidX) are merged automatically; these cover the app's own code.

# Shrink and optimize, but keep names so crash traces from users stay readable without a mapping file.
-dontobfuscate
-keepattributes SourceFile,LineNumberTable

# ---- kotlinx.serialization ----
# Annotations and nested-class metadata used by generated serializers and polymorphic lookup.
-keepattributes *Annotation*,InnerClasses,Signature,EnclosingMethod

# Generated serializers of every @Serializable class in the app and :core.
-keep,includedescriptorclasses class com.newapi.admin.**$$serializer { *; }

# Companion objects and serializer() accessors, which the runtime looks up reflectively
# (e.g. KClass.serializer() in type-safe navigation, serializer(typeOf<T>()) for generics).
-keepclassmembers class com.newapi.admin.** {
    *** Companion;
}
-keepclasseswithmembers class com.newapi.admin.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# Serializable objects are looked up through their INSTANCE field (data object routes, object cases).
-keepclassmembers @kotlinx.serialization.Serializable class com.newapi.admin.** {
    public static ** INSTANCE;
}

# Persisted session: Credentials is a sealed @Serializable interface stored as JSON with a type
# discriminator. Keep the hierarchy intact so stored sessions decode across app updates.
-keep class com.newapi.admin.core.auth.Credentials { *; }
-keep class com.newapi.admin.core.auth.Credentials$* { *; }
-keep class com.newapi.admin.core.auth.StoredAccount { *; }
-keep class com.newapi.admin.core.auth.StoredAccount$* { *; }

# Type-safe navigation routes: serial descriptors and route hash codes come from these classes.
-keep class com.newapi.admin.ui.navigation.** { *; }

# ---- OkHttp / Okio ----
# Optional TLS providers and build-time annotations that are not on Android.
-dontwarn okhttp3.internal.platform.**
-dontwarn org.conscrypt.**
-dontwarn org.bouncycastle.**
-dontwarn org.openjsse.**
-dontwarn org.codehaus.mojo.animal_sniffer.**
-dontwarn javax.annotation.**
