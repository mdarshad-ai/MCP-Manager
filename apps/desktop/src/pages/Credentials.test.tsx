import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { expect } from "vitest";
import Credentials from "./Credentials";

test("renders credentials form and handles validation and save", async () => {
  render(<Credentials />);

  // Wait for credential requirements to load
  await waitFor(() => screen.getByLabelText("API Key"));

  // Fill out the form
  fireEvent.change(screen.getByLabelText("API Key"), {
    target: { value: "secret_1234567890123456789012345678901234567890" },
  });

  // Validate credentials
  fireEvent.click(screen.getByText("Test Connection"));
  await waitFor(() => {
    expect(screen.getByText("Credentials are valid!")).toBeInTheDocument();
  });

  // Save credentials
  fireEvent.click(screen.getByText("Save"));
  await waitFor(() => screen.getByText("Credentials saved successfully!"));
});
