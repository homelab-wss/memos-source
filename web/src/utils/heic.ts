/**
 * HEIC/HEIF helpers for files that have not been uploaded yet. Most browsers
 * cannot decode these, so the editor converts a preview to JPEG while the
 * original file still uploads unchanged: the server performs the stored
 * conversion.
 */

export const isHEICFile = (file: File): boolean => {
  const mimeType = file.type.toLowerCase();
  return mimeType === "image/heic" || mimeType === "image/heif" || /\.(heic|heif)$/i.test(file.name);
};

/**
 * HEIC files picked in some browsers carry no MIME type, which would sort
 * them as documents instead of images. Fill it in from the extension; the
 * original bytes and name are unchanged.
 */
export const normalizeHEICFile = (file: File): File => {
  if (file.type || !/\.(heic|heif)$/i.test(file.name)) {
    return file;
  }
  return new File([file], file.name, { type: "image/heic", lastModified: file.lastModified });
};

/**
 * Converts a HEIC/HEIF blob to JPEG, or reports that it could not be decoded
 * so the caller can keep its original fallback. The decoder is loaded only
 * when a HEIC file actually needs it.
 */
export const convertHEICToJPEG = async (blob: Blob): Promise<Blob | undefined> => {
  try {
    const { heicTo } = await import("heic-to");
    return await heicTo({ blob, type: "image/jpeg", quality: 0.92 });
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn("Failed to convert HEIC preview; falling back to the original file.", error);
    }
    return undefined;
  }
};
