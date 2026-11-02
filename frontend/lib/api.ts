const baseUrl = process.env.BACKEND_API_URL || "http://localhost:8080";

export async function fetchFromBackend(
  endpoint: string,
  options: RequestInit & { parseAs?: "json" | "text" } = {},
) {
  const url = `${baseUrl}/${endpoint}`;

  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        // You can attach secure server-side API keys or authorization tokens here safely
        // "Authorization": `Bearer ${process.env.BACKEND_SECRET_TOKEN}`,
        ...options?.headers,
      },
    });

    if (!response.ok) {
      throw new Error(`Backend responded with status: ${response.status}`);
    }

    // Handle plain text responses (e.g., logs)
    const { parseAs } = options;
    if (parseAs === "text") {
      return response.text();
    }

    return await response.json();
  } catch (error) {
    console.error(`[Backend API Error] Failed to fetch ${endpoint}:`, error);
    throw error;
  }
}
