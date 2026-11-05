declare module 'seek-bzip' {
	const Bunzip: {
		decode(
			input: {
				readByte(): number;
				read(buffer: Uint8Array, offset: number, length: number): number;
			},
			output: { writeByte(byte: number): void }
		): void;
	};
	export default Bunzip;
}
