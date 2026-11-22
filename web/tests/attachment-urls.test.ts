import { describe, expect, it } from "vitest";
import type { Attachment } from "@/types/proto/api/attachment_service_pb";
import { getAttachmentDisplayUrl, getAttachmentMotionClipUrl, getAttachmentThumbnailUrl, getAttachmentUrl } from "@/utils/attachment";

const origin = window.location.origin;

const baseAttachment = {
  name: "attachments/test-uid",
  filename: "photo.png",
  type: "image/png",
} as Attachment;

describe("attachment filename URL paths", () => {
  const builders = [
    { name: "download", build: getAttachmentUrl, query: [] },
    { name: "thumbnail", build: getAttachmentThumbnailUrl, query: [["thumbnail", "true"]] },
    { name: "motion clip", build: getAttachmentMotionClipUrl, query: [["motion", "true"]] },
  ];

  for (const { name, build, query } of builders) {
    it.each(["100% complete.txt", "ảnh#1?.png", "literal%20name.txt"])(`keeps %s as a filename in the ${name} URL`, (filename) => {
      const url = new URL(build({ ...baseAttachment, filename } as Attachment));

      expect(decodeURIComponent(url.pathname.split("/").at(-1)!)).toBe(filename);
      expect(url.hash).toBe("");
      expect([...url.searchParams.entries()]).toEqual(query);
    });
  }
});

// Regression tests for #6128: share-mode thumbnails/motion clips must keep the share token on externalLink.
describe("attachment URL builders in share mode", () => {
  it("appends thumbnail=true to an externalLink that carries a share token", () => {
    const attachment = {
      ...baseAttachment,
      externalLink: `${origin}/file/attachments/test-uid/photo.png?share_token=abc123`,
    } as Attachment;

    const url = new URL(getAttachmentThumbnailUrl(attachment));
    expect(url.searchParams.get("thumbnail")).toBe("true");
    expect(url.searchParams.get("share_token")).toBe("abc123");
  });

  it("appends motion=true to an externalLink that carries a share token", () => {
    const attachment = {
      ...baseAttachment,
      externalLink: `${origin}/file/attachments/test-uid/photo.png?share_token=abc123`,
    } as Attachment;

    const url = new URL(getAttachmentMotionClipUrl(attachment));
    expect(url.searchParams.get("motion")).toBe("true");
    expect(url.searchParams.get("share_token")).toBe("abc123");
  });

  it("keeps the server thumbnail URL when externalLink has no share token", () => {
    const attachment = {
      ...baseAttachment,
      externalLink: "https://cdn.example.com/photo.png?version=xyz",
    } as Attachment;

    expect(getAttachmentThumbnailUrl(attachment)).toBe(`${origin}/file/attachments/test-uid/photo.png?thumbnail=true`);
  });

  it("keeps the server thumbnail URL when no externalLink is set", () => {
    expect(getAttachmentThumbnailUrl(baseAttachment)).toBe(`${origin}/file/attachments/test-uid/photo.png?thumbnail=true`);
  });

  it("leaves getAttachmentUrl returning the externalLink verbatim", () => {
    const attachment = {
      ...baseAttachment,
      externalLink: `${origin}/file/attachments/test-uid/photo.png?share_token=abc123`,
    } as Attachment;

    expect(getAttachmentUrl(attachment)).toBe(attachment.externalLink);
  });

  it("asks for a JPEG display form of a share-link HEIC and keeps the share token", () => {
    const attachment = {
      ...baseAttachment,
      type: "image/heic",
      filename: "ảnh#1?.heic",
      externalLink: `${origin}/file/attachments/test-uid/photo.heic?share_token=abc123`,
    } as Attachment;

    const url = new URL(getAttachmentDisplayUrl(attachment));
    expect(url.searchParams.get("format")).toBe("jpeg");
    expect(url.searchParams.get("share_token")).toBe("abc123");
  });
});

describe("attachment display URLs", () => {
  it.each(["image/heic", "image/heif", "IMAGE/HEIC"])(`requests the converted JPEG for %s`, (type) => {
    const attachment = { ...baseAttachment, type, filename: "IMG_0001.HEIC" } as Attachment;

    const url = new URL(getAttachmentDisplayUrl(attachment));
    expect(url.pathname).toBe(`/file/${attachment.name}/IMG_0001.HEIC`);
    expect(url.searchParams.get("format")).toBe("jpeg");
    // The original stays available for downloads.
    expect(getAttachmentUrl(attachment)).not.toContain("format=");
  });

  it("leaves other image types on the original URL", () => {
    expect(getAttachmentDisplayUrl(baseAttachment)).toBe(getAttachmentUrl(baseAttachment));
  });

  it("does not add a display parameter to an external attachment on another origin", () => {
    const attachment = {
      ...baseAttachment,
      type: "image/heic",
      externalLink: "https://cdn.example.com/photo.heic",
    } as Attachment;

    expect(getAttachmentDisplayUrl(attachment)).toBe(attachment.externalLink);
  });
});
