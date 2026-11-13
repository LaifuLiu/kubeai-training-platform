import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import Home from "./page";

describe("Home page", () => {
  it("renders the home screen heading", () => {
    render(<Home />);

    expect(screen.getByText("Home")).toBeInTheDocument();
  });
});
