export const validManifest = (path = "server.properties") => ({
  version: 1,
  server: {
    family: "minecraft",
    displayName: "Minecraft Server",
    appVersion: "1.21.7",
  },
  files: [
    {
      path,
      format: "java-properties",
      label: path,
      sections: [
        {
          id: "players",
          label: "Players",
          fields: [
            {
              key: "max-players",
              label: "Max players",
              description: "Maximum player count.",
              documentation: "https://minecraft.wiki/w/Server.properties",
              type: "integer",
              min: 1,
              max: 1000,
            },
          ],
        },
      ],
    },
  ],
});

export const duplicateManifest = () => {
  const manifest = validManifest();
  return { ...manifest, files: [manifest.files[0], manifest.files[0]] };
};

export const duplicateFieldManifest = () => {
  const manifest = validManifest();
  const file = manifest.files[0]!;
  const field = file.sections[0]!.fields[0]!;
  return {
    ...manifest,
    files: [
      {
        ...file,
        sections: [
          {
            ...file.sections[0],
            fields: [field, { ...field, label: "Duplicate" }],
          },
        ],
      },
    ],
  };
};

export const versionedFile = () => {
  const file = validManifest().files[0]!;
  const field = file.sections[0]!.fields[0]!;
  return {
    ...file,
    sections: [
      {
        id: "versions",
        label: "Versions",
        fields: [
          { ...field, key: "always" },
          { ...field, key: "legacy", until: "1.20.4" },
          { ...field, key: "modern", since: "1.21.1" },
        ],
      },
    ],
  };
};
