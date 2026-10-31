import { NextResponse } from "next/server";
import { fetchFromBackend } from "@/lib/api";

export async function GET() {
  try {
    // This calls your backend: e.g., http://localhost:8080/api/v1/training-runs
    const res = await fetchFromBackend("api/v1/training-runs");

    return NextResponse.json(res);
  } catch (error) {
    return NextResponse.json(
      { error: "Internal Server Error" },
      { status: 500 },
    );
  }
}
