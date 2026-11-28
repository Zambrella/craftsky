import { Writer } from '@zip.js/zip.js';

/** Allocate only the declared bounded entry length and reject mismatched output. */
export class ExactZipEntryWriter extends Writer<Uint8Array<ArrayBuffer>> {
	private bytes: Uint8Array<ArrayBuffer> = new Uint8Array();
	private offset = 0;

	constructor(private maximum: number) {
		super();
	}

	override async init(size = 0) {
		if (!Number.isSafeInteger(size) || size < 0 || size > this.maximum)
			throw new Error('entryLimit');
		this.bytes = new Uint8Array(size);
		await super.init?.(size);
	}

	override async writeUint8Array(chunk: Uint8Array) {
		if (this.offset + chunk.length > this.bytes.length)
			throw new Error('invalidArchive');
		this.bytes.set(chunk, this.offset);
		this.offset += chunk.length;
	}

	override async getData() {
		if (this.offset !== this.bytes.length) throw new Error('invalidArchive');
		return this.bytes;
	}
}
