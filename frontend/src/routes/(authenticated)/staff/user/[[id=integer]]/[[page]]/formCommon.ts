export const isError = (value: string | undefined) => {
	return value != undefined;
};

export const labelColor = (value: string | undefined) => {
	if (isError(value)) return 'red';
	return 'gray';
};

export const inputColor = (value: string | undefined) => {
	if (isError(value)) return 'red';
	return 'base';
};
