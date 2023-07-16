<script lang="ts">
  import { Center, Notification, Grid, Space, Text, ThemeIcon, Timeline, Stack, Modal, Group, Button, Input, SimpleGrid } from "@svelteuidev/core";
  import type { UserInfo } from "./userCard";

  import TimelineItemBody from "./TimelineItemBody.svelte";
  import type { Training } from "$lib/src/types";

  import { createEventDispatcher } from 'svelte';
	import Timestamp from "$lib/components/Timestamp.svelte";

	const dispatch = createEventDispatcher();

  export let userInfo: UserInfo;

    const closeRemoveTrainingModal = () => {
      removeTrainingModal = {
        active: undefined,
        open: false
      };
    };

    let removeTrainingModal: {
      active: Training | undefined,
      open: boolean
    } = {
      active: undefined,
      open: false
    };

    const closeCreateTrainingModal = () => {
      createTrainingModal = {
        open: false,
        value: ''
      };
    };

    let createTrainingModal: {
      open: boolean,
      value: string
    } = {
      open: false,
      value: ''
    };


    let refeshUserTraining = () => dispatch('refresh');
</script>

<Grid cols={9}>
  <Grid.Col md={6} lg={3}>
    <Center>
      <Text>User Info</Text>
    </Center>
    <Space h="lg" />
    <Stack>
    <SimpleGrid cols={2}>
      <Text color="dimmed">Name</Text>
      <Text>{userInfo.user.name}</Text>

      <Text color="dimmed">ID</Text>
      <Text>{userInfo.user.id}</Text>

      <Text color="dimmed">Email</Text>
      <Text>{userInfo.user.email}</Text>

      <Text color="dimmed">Join Date</Text>
      <Text><Timestamp time={userInfo.user.createdAt} /></Text>

      <Text color="dimmed">Last Updated</Text>
      <Text><Timestamp time={userInfo.user.updatedAt} /></Text>

      <Text color="dimmed">Enabled</Text>
      <Text>{userInfo.user.enabled ? 'Yes' : 'No'}</Text>

      <Text color="dimmed">Admin</Text>
      <Text>{userInfo.user.admin ? 'Yes' : 'No'}</Text>

      <Text color="dimmed">Role</Text>
      <Text>{userInfo.user.role}</Text>

      <Text color="dimmed">User Type</Text>
      <Text>{userInfo.user.type}</Text>

      {#if userInfo.user.graduationYear > 0}
      <Text color="dimmed">Graduation Year</Text>
      <Text>{userInfo.user.graduationYear}</Text>
      {/if}

      {#if userInfo.user.major}
      <Text color="dimmed">Major</Text>
      <Text>{userInfo.user.major}</Text>
      {/if}
    </SimpleGrid>
    <Space h="lg" />
    <Button color="blue" fullSize href={"/admin/user?id=" + userInfo.user.id}>
      Edit User
    </Button>
  </Stack>
</Grid.Col>

    <Grid.Col md={6} lg={3}>
      <Center>
        <Text>User History</Text>
      </Center>
      <Space h="lg" />
      <Timeline active={userInfo.timelineItems.length} bulletSize={24} lineWidth={2}>
          {#each userInfo.timelineItems as item}
            <Timeline.Item title={item.getTitle()}>
              <svelte:fragment slot='bullet'>
                <ThemeIcon radius='xl' color={item.getBulletColor()}><svelte:component this={item.getBullet()} /> </ThemeIcon>
              </svelte:fragment>
              <TimelineItemBody timelineItem={item} />
          </Timeline.Item>
          {/each}
      </Timeline>
  </Grid.Col>
    <Grid.Col md={6} lg={3}>
        <Center>
          <Text>User Trainings</Text>
        </Center>
        <Space h="lg" />
        <Stack>
          <Modal opened={removeTrainingModal.open} title="Remove Training" centered on:close={closeRemoveTrainingModal}>
            Are you sure you want to remove {removeTrainingModal.active?.trainingType} from {userInfo.user.name}?
            <Space h="lg" />
            <Group position="center">
                <Button on:click={closeRemoveTrainingModal}>
                  Cancel
                </Button>
                <Space w="lg" />
                <Button color="red" on:click={async () => {
                  await removeTrainingModal.active?.remove();
                  closeRemoveTrainingModal();
                  refeshUserTraining();
                }}>
                  Remove
                </Button>
            </Group>
          </Modal>
          <Modal opened={createTrainingModal.open} title="Create New Training" centered on:close={closeCreateTrainingModal}>
            Create a training for {userInfo.user.name}
            <Space h="lg" />
            <Text color="dimmed">Training Type</Text>
            <Space h="sm" />
            <Input bind:value={createTrainingModal.value}/>
            <Space h="lg" />
            <Group position="center">
                <Button on:click={closeCreateTrainingModal}>
                  Cancel
                </Button>
                <Space w="lg" />
                <Button color="green" on:click={async () => {
                  await userInfo.user.createTraining(createTrainingModal.value.toLowerCase());
                  closeCreateTrainingModal();
                  refeshUserTraining();
                }}>
                  Create
                </Button>
            </Group>
          </Modal>
          <Button color="green" fullSize on:click={() => {
            createTrainingModal = {
              open: true,
              value: ''
            };
          }}>
            Create Training
          </Button>
          {#each userInfo.trainings as training}
            {#if training.deletedAt === undefined}
              <Notification  title={training.trainingType} color="blue" withCloseButton={true} on:close={() => {
                removeTrainingModal = {
                  active: training,
                  open: true
                };
              }}>
              </Notification>
            {:else}
              <Notification  title={training.trainingType} color="gray" withCloseButton={false}>
              </Notification>
            {/if}
          {/each}
        </Stack>
    </Grid.Col>
</Grid>