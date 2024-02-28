<script lang="ts">
  import { Map, NavigationControl, Popup } from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import { env } from '$env/dynamic/public';

  const tileKey = env.PUBLIC_TILE_API_KEY;

  let center = [-72.53111867309333, 42.39224330004294] as [number, number];

  let map: Map;

  function createMap(container: HTMLElement): Map {
    let map = new Map({
        container,
        style: `https://api.maptiler.com/maps/bright/style.json?key=${tileKey}`,
        center,
        zoom: 17
    });

    map.addControl(new NavigationControl());

    map.on('load', async () => {
        const image = await map.loadImage('https://maplibre.org/maplibre-gl-js/docs/assets/custom_marker.png');
        // Add an image to use as a custom marker
        map.addImage('custom-marker', image.data);

        map.addSource('places', {
            'type': 'geojson',
            'data': {
                'type': 'FeatureCollection',
                'features': [
                    {
                        'type': 'Feature',
                        'properties': {
                            'description':
                                '<strong>UMass Amherst Makerspace</strong><p>Agricultural Engineering Building North, rooms 120&122, Amherst, MA 01003</p>'
                        },
                        'geometry': {
                            'type': 'Point',
                            'coordinates': center
                        }
                    },
                ]
            }
        });

        // Add a layer showing the places.
        map.addLayer({
            'id': 'places',
            'type': 'symbol',
            'source': 'places',
            'layout': {
                'icon-image': 'custom-marker',
                'icon-overlap': 'always'
            }
        });

        // Create a popup, but don't add it to the map yet.
        const popup = new Popup({
            closeButton: false,
            closeOnClick: false
        });

        map.on('mouseenter', 'places', (e) => {
            // Change the cursor style as a UI indicator.
            map.getCanvas().style.cursor = 'pointer';

            if (!e.features || !e.features.length) {
                return;
            }

            const geom = e.features[0].geometry;
            if (!("coordinates" in geom)) {
                return;
            }
            

            const coordinates = geom.coordinates.slice();
            if (typeof coordinates[0] !== 'number' || typeof coordinates[1] !== 'number') {
                return;
            }

            const description = e.features[0].properties.description;

            // Ensure that if the map is zoomed out such that multiple
            // copies of the feature are visible, the popup appears
            // over the copy being pointed to.
            while (Math.abs(e.lngLat.lng - coordinates[0]) > 180) {
                coordinates[0] += e.lngLat.lng > coordinates[0] ? 360 : -360;
            }

            // Populate the popup and set its coordinates
            // based on the feature found.
            popup.setLngLat(coordinates as [number, number]).setHTML(description).addTo(map);
        });

        map.on('mouseleave', 'places', () => {
            map.getCanvas().style.cursor = '';
            popup.remove();
        });
    });


    return map;
  }

  function mapAction(container: HTMLElement) {
    map = createMap(container);
    return {
      destroy: () => {
        map.remove();
      },
    };
  }
</script>

<style>
  .container {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .map {
    aspect-ratio: 16/9;
  }
</style>

<div class="container">
  <div class="map flex-1" use:mapAction />
</div>