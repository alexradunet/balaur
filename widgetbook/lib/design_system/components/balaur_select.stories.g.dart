// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_select.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurSelect, StoryArgs<BalaurSelect>>;
typedef _Scenario = BalaurSelectScenario;
typedef _Defaults = BalaurSelectDefaults;
typedef _Story = BalaurSelectStory;
typedef _Args = BalaurSelectArgs;
final BalaurSelectComponent = Component<BalaurSelect, StoryArgs<BalaurSelect>>(
  name: component.name ?? 'BalaurSelect',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a labelled parchment selector.''',
  stories: [
    $Model..$generatedName = 'Model',
    $Disabled..$generatedName = 'Disabled',
  ],
);
typedef BalaurSelectScenario = Scenario<BalaurSelect, BalaurSelectArgs>;
typedef BalaurSelectDefaults = Defaults<BalaurSelect, BalaurSelectArgs>;

class BalaurSelectStory extends Story<BalaurSelect, BalaurSelectArgs> {
  BalaurSelectStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    required super.args,
    StoryWidgetBuilder<BalaurSelect, BalaurSelectArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         builder:
             builder ??
             (context, args) => BalaurSelect(
               key: args.key,
               options: args.options,
               value: args.value,
               onChanged: args.onChanged,
               label: args.label,
               enabled: args.enabled,
             ),
       );
}

class BalaurSelectArgs extends StoryArgs<BalaurSelect> {
  BalaurSelectArgs({
    Arg<Key?>? key,
    required Arg<List<String>> options,
    Arg<String>? value,
    Arg<void Function(String?)?>? onChanged,
    Arg<String?>? label,
    Arg<bool>? enabled,
  }) : this.keyArg = $initArg('key', key, null),
       this.optionsArg = $initArg('options', options, null)!,
       this.valueArg = $initArg('value', value, StringArg(''))!,
       this.onChangedArg = $initArg('onChanged', onChanged, null),
       this.labelArg = $initArg('label', label, NullableStringArg(null))!,
       this.enabledArg = $initArg('enabled', enabled, BoolArg(true))!;

  BalaurSelectArgs.fixed({
    Key? key,
    required List<String> options,
    String value = '',
    void Function(String?)? onChanged,
    String? label = null,
    bool enabled = true,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.optionsArg = $initArg('options', Arg.fixed(options), null)!,
       this.valueArg = $initArg('value', Arg.fixed(value), null)!,
       this.onChangedArg = $initArg(
         'onChanged',
         onChanged == null ? null : Arg.fixed(onChanged),
         null,
       ),
       this.labelArg = $initArg(
         'label',
         label == null ? null : Arg.fixed(label),
         null,
       ),
       this.enabledArg = $initArg('enabled', Arg.fixed(enabled), null)!;

  final Arg<Key?>? keyArg;

  final Arg<List<String>> optionsArg;

  final Arg<String> valueArg;

  final Arg<void Function(String?)?>? onChangedArg;

  final Arg<String?>? labelArg;

  final Arg<bool> enabledArg;

  Key? get key => keyArg?.value;

  List<String> get options => optionsArg.value;

  String get value => valueArg.value;

  void Function(String?)? get onChanged => onChangedArg?.value;

  String? get label => labelArg?.value;

  bool get enabled => enabledArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    optionsArg,
    valueArg,
    onChangedArg,
    labelArg,
    enabledArg,
  ];
}
