// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format width=80

// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_import, prefer_relative_imports, directives_ordering, unused_element, strict_raw_type

part of 'balaur_toggle.stories.dart';

// **************************************************************************
// StoryGenerator
// **************************************************************************

typedef _Component = Component<BalaurToggle, StoryArgs<BalaurToggle>>;
typedef _Scenario = BalaurToggleScenario;
typedef _Defaults = BalaurToggleDefaults;
typedef _Story = BalaurToggleStory;
typedef _Args = BalaurToggleArgs;
final BalaurToggleComponent = Component<BalaurToggle, StoryArgs<BalaurToggle>>(
  name: component.name ?? 'BalaurToggle',
  path: component.path ?? 'design_system/components',
  docsBuilder: component.docsBuilder,
  docComment: r'''Displays a square Hearthwood toggle.''',
  stories: [
    $Off..$generatedName = 'Off',
    $On..$generatedName = 'On',
    $Disabled..$generatedName = 'Disabled',
  ],
);
typedef BalaurToggleScenario = Scenario<BalaurToggle, BalaurToggleArgs>;
typedef BalaurToggleDefaults = Defaults<BalaurToggle, BalaurToggleArgs>;

class BalaurToggleStory extends Story<BalaurToggle, BalaurToggleArgs> {
  BalaurToggleStory({
    super.name,
    super.designLink,
    super.setup,
    super.modes,
    BalaurToggleArgs? args,
    StoryWidgetBuilder<BalaurToggle, BalaurToggleArgs>? builder,
    super.scenarios,
    super.excludeFromTests,
  }) : super(
         args: args ?? BalaurToggleArgs(),
         builder:
             builder ??
             (context, args) => BalaurToggle(
               key: args.key,
               checked: args.checked,
               onChanged: args.onChanged,
               label: args.label,
               enabled: args.enabled,
             ),
       );
}

class BalaurToggleArgs extends StoryArgs<BalaurToggle> {
  BalaurToggleArgs({
    Arg<Key?>? key,
    Arg<bool>? checked,
    Arg<void Function(bool)?>? onChanged,
    Arg<String?>? label,
    Arg<bool>? enabled,
  }) : this.keyArg = $initArg('key', key, null),
       this.checkedArg = $initArg('checked', checked, BoolArg(false))!,
       this.onChangedArg = $initArg('onChanged', onChanged, null),
       this.labelArg = $initArg('label', label, NullableStringArg(null))!,
       this.enabledArg = $initArg('enabled', enabled, BoolArg(true))!;

  BalaurToggleArgs.fixed({
    Key? key,
    bool checked = false,
    void Function(bool)? onChanged,
    String? label = null,
    bool enabled = true,
  }) : this.keyArg = $initArg('key', key == null ? null : Arg.fixed(key), null),
       this.checkedArg = $initArg('checked', Arg.fixed(checked), null)!,
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

  final Arg<bool> checkedArg;

  final Arg<void Function(bool)?>? onChangedArg;

  final Arg<String?>? labelArg;

  final Arg<bool> enabledArg;

  Key? get key => keyArg?.value;

  bool get checked => checkedArg.value;

  void Function(bool)? get onChanged => onChangedArg?.value;

  String? get label => labelArg?.value;

  bool get enabled => enabledArg.value;

  @override
  List<Arg?> get list => [
    keyArg,
    checkedArg,
    onChangedArg,
    labelArg,
    enabledArg,
  ];
}
