/*
The generator package generates kubernetes objects from a "compose" file and transforms them into a helm chart.

The generator package is the core of katenary. It is responsible for generating kubernetes objects from a compose file and transforming them into a helm chart.
Convertion manipulates Yaml representation of kubernetes object to add conditions, labels, annotations, etc. to the objects. It also create the values to be set to
the values.yaml file.

The generate.Convert() create an HelmChart object and call "Generate()" method to convert from a compose file to a helm chart.
It saves the helm chart in the given directory.

If you want to change or override the write behavior, you can use the HelmChart.Generate() function and implement your own write function. This function returns
the helm chart object containing all kubernetes objects and helm chart ingormation. It does not write the helm chart to the disk.
*/
package generator
