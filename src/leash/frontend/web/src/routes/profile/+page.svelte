<script lang="ts">
	import { User, type UserUpdateOptions } from '$lib/src/leash';
	import { Alert, Button, Center, NativeSelect, NumberInput, Paper, Seo, Skeleton, Stack, Tabs, TextInput } from '@svelteuidev/core';
	import { CrossCircled } from 'radix-icons-svelte';

	let k = {};

	let user: User;

	let name: string;
	let email: string;
	let accountType: string;
	let major: string;
	let graduationYear: number;

	let modified = false;

	function resetUserData() {
		name = user.name;
		email = user.email;
		accountType = user.type;
		major = user.major;
		graduationYear = user.graduationYear;
		modified = false;
	}

	async function getUser() {
		user = await User.self();
		resetUserData();
	}

	function modify() {
		modified = true;
	}

	function save() {
		modified = false;
		let req: UserUpdateOptions = {};

		if (name != user.name) {
			req.name = name;
		}

		if (accountType != user.type) {
			req.type = accountType;
		}

		if (major != user.major) {
			req.major = major;
		}

		if (graduationYear != user.graduationYear) {
			req.graduationYear = graduationYear;
		}
		
		user.update(req).then((user) => {
			user = user;
			resetUserData();
		});
	}
</script>

<Seo title="Profile" description="Modify your own information." />

{#await getUser()}
	<Skeleton />
{:then}
	<Tabs on:change={resetUserData}>
		<Tabs.Tab label="Account">
			<Center>
				<Paper>
					<div class="fill">
						<Stack>
							<TextInput label="Full Name" bind:value={name} on:change={modify} on:input={modify}/>
							<TextInput disabled label="Email" value={email}/>
							<Button disabled={!modified} color="blue" variant="filled" fullSize on:click={save}>Save</Button>
						</Stack>
					</div>
				</Paper>
			</Center>
		</Tabs.Tab>
		<Tabs.Tab label="Education">
			<Center>
				<Paper>
					<div class="fill">
						<Stack>
							<NativeSelect
								data={[
									{ value: 'undergrad', label: 'Undergrad Student' },
									{ value: 'grad', label: 'Graduate Student' },
									{ value: 'faculty', label: 'Faculty' },
									{ value: 'staff', label: 'Makerspace Staff' },
									{ value: 'alumni', label: 'Alumni' },
									{ value: 'other', label: 'Other' }
								]}
								value={accountType}
								disabled
								label="Account Type"
							/>
							<TextInput label="Major" bind:value={major} on:change={modify} on:input={modify}/>
							<NumberInput label="Graduation Year" bind:value={graduationYear} on:change={modify}/>
							<Button disabled={!modified} color="blue" variant="filled" fullSize on:click={save}>Save</Button>
						</Stack>
					</div>
				</Paper>
			</Center>
		</Tabs.Tab>
	</Tabs>
{:catch error}
	<Alert icon={CrossCircled} title="Error" color="red" variant="filled">
		{error.message}
	</Alert>
{/await}

<style lang="scss">
	.fill {
		min-width: 50dvw;
	}
</style>
