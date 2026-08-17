package com.sub2api.app;

import java.net.HttpURLConnection;
import java.net.URL;

final class SourceRegistry {
    static final String PROBE_PATH = "health";
    static final String UPDATE_MANIFEST_PATH = "api/artifacts/update/android/sub2api";
    static final String[] URLS = {
        "http://192.168.3.2:18381/",
        "http://fpsq.xyz:18381/",
    };
    static final String[] UPDATE_URLS = {
        "http://192.168.3.2:11111/",
        "http://fpsq.xyz:11112/",
    };
    static final String[] LABELS = {
        "局域网",
        "公网直连",
    };

    private SourceRegistry() {}

    static boolean isValidIndex(int index) {
        return index >= 0 && index < URLS.length;
    }

    static ProbeResult probe(int index) {
        long startedAt = System.nanoTime();
        HttpURLConnection connection = null;
        try {
            connection = (HttpURLConnection) new URL(URLS[index] + PROBE_PATH).openConnection();
            connection.setRequestMethod("GET");
            connection.setConnectTimeout(2500);
            connection.setReadTimeout(5000);
            connection.setInstanceFollowRedirects(false);
            connection.setUseCaches(false);
            int status = connection.getResponseCode();
            long latencyMs = Math.max(1, (System.nanoTime() - startedAt) / 1_000_000);
            return new ProbeResult(index, status >= 200 && status < 300, latencyMs, status);
        } catch (Exception error) {
            long latencyMs = Math.max(1, (System.nanoTime() - startedAt) / 1_000_000);
            return new ProbeResult(index, false, latencyMs, 0);
        } finally {
            if (connection != null) {
                connection.disconnect();
            }
        }
    }

    static final class ProbeResult {
        final int index;
        final boolean healthy;
        final long latencyMs;
        final int statusCode;

        ProbeResult(int index, boolean healthy, long latencyMs, int statusCode) {
            this.index = index;
            this.healthy = healthy;
            this.latencyMs = latencyMs;
            this.statusCode = statusCode;
        }
    }
}
