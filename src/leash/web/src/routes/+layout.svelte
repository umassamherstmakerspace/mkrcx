<script lang="ts">
	import HeadContent from "$lib/components/HeadContent.svelte";
	import { AppShell, Header, Title, UnstyledButton } from "@svelteuidev/core";
	import { quadIn, quadOut } from "svelte/easing";
	import { fade, fly, slide } from "svelte/transition";

    let menu = false;
    const toggleMenu = () => {
        menu = !menu;
    }

    $: if (menu) {
        openMenu();
    } else {
        closeMenu();
    }

    let prevBodyPosition: string;
    let prevBodyOverflow: string;
    let prevBodyWidth: string;
    let scrollY: number;


    const openMenu = () => {
        scrollY = document.scrollingElement?.scrollTop || 0;
        prevBodyPosition = document.body.style.position;
        prevBodyOverflow = document.body.style.overflow;
        prevBodyWidth = document.body.style.width;
        document.body.style.top = `-${scrollY}px`;
        document.body.style.overflow = 'hidden';
        menu = true;
  };

  const closeMenu = () => {
    document.body.style.position = prevBodyPosition || '';
    document.body.style.top = '';
    document.body.style.overflow = prevBodyOverflow || '';
    document.body.style.width = prevBodyWidth || '';
    window.scrollTo(0, scrollY);
    menu = false;
  };

</script>

<AppShell>
    <div class="sticky">
        <Header height={80} slot="header">
            <HeadContent bind:menu/>
        </Header>
    </div>
        {#if menu}
            <div class="fullscreen dimmed" on:click={toggleMenu} on:keydown={toggleMenu} role="dialog" aria-modal="true" aria-hidden="true">
                <div class="tall" in:slide={{ duration: 300, easing: quadIn, axis: "x" }} out:slide={{ duration: 200, easing: quadOut, axis: "x" }}>
                    <Title>Menu</Title>
                </div>
            </div>
        {/if}
    <slot />
</AppShell>


<style>
    .fullscreen{
    position:fixed;
    top:0;
    left:0;
    bottom:0;
    right:0;
    height:100%;
    width:100%;
    z-index:1000;
    overflow:hidden;
}

.dimmed {
    background-color: rgba(0,0,0,0.5);
}

.tall {
    height: 100%;
    display: inline-block;
    background-color: white;
}

.sticky {
    position: sticky;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 100;
}
</style>