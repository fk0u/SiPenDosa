package com.sipendosa.app;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.provider.OpenableColumns;

import java.io.File;
import java.io.FileNotFoundException;

/**
 * ApkFileProvider provides secure content:// URI access for Android Package Installer
 * without needing third-party libraries or AndroidX dependencies.
 */
public class ApkFileProvider extends ContentProvider {
    public static final String AUTHORITY = "com.sipendosa.app.fileprovider";

    @Override
    public boolean onCreate() {
        return true;
    }

    private File getFileForUri(Uri uri) throws FileNotFoundException {
        if (getContext() == null) {
            throw new FileNotFoundException("Provider context is null");
        }

        String path = uri.getPath();
        if (path == null) {
            throw new FileNotFoundException("Null URI path");
        }

        String fileName = uri.getLastPathSegment();
        if (fileName == null) {
            throw new FileNotFoundException("Null file segment");
        }

        // 1. Cek External Files Directory (App-specific, no storage permission required)
        File extDir = getContext().getExternalFilesDir(null);
        if (extDir != null) {
            File target = new File(extDir, fileName);
            if (target.exists()) {
                return target;
            }
        }

        // 2. Cek Cache Directory
        File cacheDir = getContext().getCacheDir();
        if (cacheDir != null) {
            File target = new File(cacheDir, fileName);
            if (target.exists()) {
                return target;
            }
        }

        throw new FileNotFoundException("Update APK file not found for URI: " + uri);
    }

    @Override
    public ParcelFileDescriptor openFile(Uri uri, String mode) throws FileNotFoundException {
        File file = getFileForUri(uri);
        return ParcelFileDescriptor.open(file, ParcelFileDescriptor.MODE_READ_ONLY);
    }

    @Override
    public Cursor query(Uri uri, String[] projection, String selection, String[] selectionArgs, String sortOrder) {
        String[] columns = projection != null ? projection : new String[]{
                OpenableColumns.DISPLAY_NAME,
                OpenableColumns.SIZE
        };

        MatrixCursor cursor = new MatrixCursor(columns);
        try {
            File file = getFileForUri(uri);
            Object[] row = new Object[columns.length];
            for (int i = 0; i < columns.length; i++) {
                if (OpenableColumns.DISPLAY_NAME.equals(columns[i])) {
                    row[i] = file.getName();
                } else if (OpenableColumns.SIZE.equals(columns[i])) {
                    row[i] = file.length();
                } else {
                    row[i] = null;
                }
            }
            cursor.addRow(row);
        } catch (FileNotFoundException ignored) {}

        return cursor;
    }

    @Override
    public String getType(Uri uri) {
        return "application/vnd.android.package-archive";
    }

    @Override
    public Uri insert(Uri uri, ContentValues values) {
        return null;
    }

    @Override
    public int delete(Uri uri, String selection, String[] selectionArgs) {
        return 0;
    }

    @Override
    public int update(Uri uri, ContentValues values, String selection, String[] selectionArgs) {
        return 0;
    }
}
