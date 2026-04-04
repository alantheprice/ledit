# Add project specific ProGuard rules here.

# Keep the ledit mobile bindings
-keep class com.ledit.editor.** { *; }

# Keep NanoHttpd
-keep class fi.iki.elonen.** { *; }
-dontwarn fi.iki.elonen.**

# Keep Android generated classes
-keep class * extends android.app.Service
-keep class * extends androidx.fragment.app.Fragment

# Kotlin
-keep class kotlin.** { *; }
-keep class kotlin.Metadata { *; }
-dontwarn kotlin.**

# Keep Parcelable
-keep class * implements android.os.Parcelable {
    public static final android.os.Parcelable$Creator *;
}