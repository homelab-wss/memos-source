import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  heicTo: vi.fn(),
  extractMetadata: vi.fn(),
}));

// The ingest helper lives beside useFileUpload, whose module graph reaches
// AuthContext and the query hooks; stub every client they name-import.
vi.mock("@/connect", () => ({
  attachmentServiceClient: {},
  authServiceClient: {},
  userServiceClient: {},
  refreshAccessToken: vi.fn(),
}));

vi.mock("@/components/MemoEditor/edit/services/mediaMetadataService", () => ({
  mediaMetadataService: { extract: mocks.extractMetadata },
}));

vi.mock("heic-to", () => ({ heicTo: mocks.heicTo }));

import { prepareLocalFiles } from "@/components/MemoEditor/edit/hooks/useFileUpload";

const createBlobUrl = (blob: Blob | File) => (blob instanceof File ? `file:${blob.name}` : "blob:converted");

describe("HEIC previews at file ingest", () => {
  it("keeps the original bytes, previews the converted JPEG, and fills in a missing MIME type", async () => {
    const file = new File(["heic"], "IMG_0001.HEIC", { type: "" });
    const converted = new Blob(["jpeg"], { type: "image/jpeg" });
    mocks.heicTo.mockResolvedValue(converted);

    const [localFile] = await prepareLocalFiles([file], { createBlobUrl, saveMediaMetadata: false });

    const [{ blob, type, quality }] = mocks.heicTo.mock.calls[0];
    expect(blob).toBe(file);
    expect(type).toBe("image/jpeg");
    expect(quality).toBe(0.92);
    expect(localFile.file.name).toBe(file.name);
    expect(localFile.file.type).toBe("image/heic");
    expect(localFile.previewUrl).toBe("blob:converted");
  });

  it("keeps the original preview when conversion fails", async () => {
    const file = new File(["heic"], "IMG_0002.heic", { type: "image/heic" });
    mocks.heicTo.mockRejectedValue(new Error("unsupported"));

    const [localFile] = await prepareLocalFiles([file], { createBlobUrl, saveMediaMetadata: false });

    expect(localFile.previewUrl).toBe("file:IMG_0002.heic");
  });

  it("does not load the converter for other images", async () => {
    const file = new File(["png"], "photo.png", { type: "image/png" });

    const [localFile] = await prepareLocalFiles([file], { createBlobUrl, saveMediaMetadata: false });

    expect(mocks.heicTo).not.toHaveBeenCalled();
    expect(localFile.previewUrl).toBe("file:photo.png");
  });
});
