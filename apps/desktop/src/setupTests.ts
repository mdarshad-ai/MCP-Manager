import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import "@testing-library/jest-dom";

const handlers = [
  http.get("http://127.0.0.1:7099/v1/credentials/:provider", ({ params }) => {
    if (params.provider === "notion") {
      return HttpResponse.json({
        provider: "notion",
        credentials: [
          {
            key: "api_key",
            displayName: "API Key",
            required: true,
            secret: true,
            validation: "^secret_[a-zA-Z0-9]{40,}$",
            description: "Notion integration API key",
          },
        ],
      });
    }
  }),
  http.post("http://127.0.0.1:7099/v1/credentials/validate", async () => {
    return HttpResponse.json({ valid: true, message: "Credentials are valid" });
  }),
  http.post("http://127.0.0.1:7099/v1/credentials", async () => {
    return HttpResponse.json({ success: true, message: "Credentials saved successfully!" });
  }),
];

const server = setupServer(...handlers);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
