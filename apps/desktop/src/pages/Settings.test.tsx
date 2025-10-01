import { render, screen, waitFor } from "@testing-library/react";
import { Settings } from "./Settings";
import * as api from "../api";

vi.mock("../api");

test("renders settings page and loads initial data", async () => {
  const mockSettings: api.AppSettings = {
    theme: "dark",
    autostart: true,
    logsCap: 500,
    performance: {
      refreshInterval: 10000,
      maxLogLines: 2000,
    },
    storage: {
      used: 1024 * 1024 * 200, // 200MB
      available: 1024 * 1024 * 800, // 800MB
    },
  };

  (api.fetchSettings as vi.Mock).mockResolvedValue(mockSettings);

  render(<Settings />);

  await waitFor(() => {
    expect(screen.getByLabelText("Autostart Manager")).toBeChecked();
  });

  expect(screen.getByLabelText("Logs Storage Limit (MB)")).toHaveValue(500);
  expect(screen.getByLabelText("Refresh Interval (seconds)")).toHaveValue(10);
});
