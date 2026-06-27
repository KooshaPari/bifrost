/**
 * Port and URL utility - single source of truth for Bifrost backend connectivity
 *
 * This utility handles:
 * - Development vs Production environment detection
 * - Dynamic port resolution
 * - URL generation for API calls and WebSocket connections
 * - Automatic protocol detection (http/https, ws/wss)
 * - Cross-origin deployments via VITE_BIFROST_API_BASE_URL
 */

interface PortConfig {
	port: string;
	isDevelopment: boolean;
	baseUrl: string;
	wsUrl: string;
	host: string;
}

function normalizeApiOrigin(origin: string): string {
	return origin.trim().replace(/\/+$/, "");
}

/**
 * Optional explicit Bifrost gateway origin for split UI/API hosting (e.g. Vercel).
 * Example: https://bifrost.example.com
 */
function getConfiguredApiOrigin(): string | null {
	const configured = process.env.VITE_BIFROST_API_BASE_URL;
	if (!configured || configured.trim() === "") {
		return null;
	}

	return normalizeApiOrigin(configured);
}

function getConfiguredPortConfig(origin: string): PortConfig {
	const url = new URL(origin);
	const isHttps = url.protocol === "https:";

	return {
		port: url.port || (isHttps ? "443" : "80"),
		isDevelopment: false,
		baseUrl: origin,
		wsUrl: `${isHttps ? "wss:" : "ws:"}//${url.host}`,
		host: url.host,
	};
}

/**
 * Gets the current port configuration based on environment
 */
function getPortConfig(): PortConfig {
	const configuredOrigin = getConfiguredApiOrigin();
	if (configuredOrigin) {
		return getConfiguredPortConfig(configuredOrigin);
	}

	const isDevelopment = process.env.NODE_ENV === "development";

	if (isDevelopment) {
		// Development mode: Vite dev server runs on different port than Go server
		const port = process.env.BIFROST_PORT || "8080";
		return {
			port,
			isDevelopment: true,
			baseUrl: `http://localhost:${port}`,
			wsUrl: `ws://localhost:${port}`,
			host: `localhost:${port}`,
		};
	} else {
		// Production mode: UI is served by the same Go server
		// Use current window location for automatic port detection
		if (typeof window !== "undefined") {
			const protocol = window.location.protocol === "https:" ? "https:" : "http:";
			const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";

			return {
				port: window.location.port || (window.location.protocol === "https:" ? "443" : "80"),
				isDevelopment: false,
				baseUrl: `${protocol}//${window.location.host}`,
				wsUrl: `${wsProtocol}//${window.location.host}`,
				host: window.location.host,
			};
		} else {
			// Server-side rendering fallback - use relative URLs
			return {
				port: "unknown",
				isDevelopment: false,
				baseUrl: "",
				wsUrl: "",
				host: "",
			};
		}
	}
}

/**
 * Get the current port number as a string
 */
export function getPort(): string {
	return getPortConfig().port;
}

/**
 * Get the base URL for API calls (includes protocol and host)
 */
export function getApiBaseUrl(): string {
	const config = getPortConfig();

	if (config.isDevelopment) {
		return `${config.baseUrl}/api`;
	}

	if (getConfiguredApiOrigin()) {
		return `${config.baseUrl}/api`;
	}

	// Production mode: use relative URL for API calls
	return "/api";
}

/**
 * Get the WebSocket URL for real-time connections
 */
export function getWebSocketUrl(path: string = ""): string {
	const config = getPortConfig();
	const cleanPath = path.startsWith("/") ? path : `/${path}`;

	return `${config.wsUrl}${cleanPath}`;
}

/**
 * Get the full base URL (for example code snippets)
 */
export function getExampleBaseUrl(): string {
	return getPortConfig().baseUrl;
}

/**
 * Get the host (hostname:port) for example code
 */
export function getExampleHost(): string {
	return getPortConfig().host;
}

/**
 * Check if we're in development mode
 */
export function isDevelopmentMode(): boolean {
	return getPortConfig().isDevelopment;
}

/**
 * Generate a complete URL for a specific endpoint
 */
export function getEndpointUrl(endpoint: string): string {
	const config = getPortConfig();
	const cleanEndpoint = endpoint.startsWith("/") ? endpoint : `/${endpoint}`;

	if (config.isDevelopment || getConfiguredApiOrigin()) {
		return `${config.baseUrl}${cleanEndpoint}`;
	}

	// Production mode: use relative URLs
	return cleanEndpoint;
}