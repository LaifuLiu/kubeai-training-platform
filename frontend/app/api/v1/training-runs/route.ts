import { NextRequest, NextResponse } from "next/server";
import { fetchFromBackend } from "@/lib/api";

export async function GET(request: NextRequest) {
  try {
    // Read pagination parameters from the frontend request
    const searchParams = request.nextUrl.searchParams;

    const page = searchParams.get("page") ?? "1";
    const size = searchParams.get("size") ?? "10";

    // Forward them to the Go backend
    const res = await fetchFromBackend(
      `api/v1/training-runs?page=${page}&size=${size}`,
    );

    return NextResponse.json(res);
  } catch (error) {
    return NextResponse.json(
      { error: "Internal Server Error" },
      { status: 500 },
    );
  }
}
